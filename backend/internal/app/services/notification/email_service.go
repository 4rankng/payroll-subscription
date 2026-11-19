package notification

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/config"
	"api-server/internal/constants"
	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"
)

// EmailService coordinates email delivery via a configurable provider.
type EmailService struct {
	cfg              config.NotificationConfig
	delivery         domain.EmailDeliveryPort
	notificationRepo domain.NotificationRepository
	userRepo         domain.UserRepository
	eventPublisher   domain.EmailEventPublisher
	assetSvc         AssetStoragePort
	logger           *slog.Logger
}

const (
	defaultEmailSender = constants.DefaultEmailSenderName + " <" + constants.DefaultEmailSenderAddress + ">"
)

var approvedEmailSenders = []string{
	defaultEmailSender,
}

// AssetStoragePort is a minimal interface for saving files as assets.
type AssetStoragePort interface {
	UploadAssetFromBytes(ctx context.Context, data []byte, filename string, uploadType string, uploadedBy uint) (*domain.Asset, error)
}

// NewEmailService constructs a new EmailService instance.
func NewEmailService(
	cfg config.NotificationConfig,
	delivery domain.EmailDeliveryPort,
	notificationRepo domain.NotificationRepository,
	userRepo domain.UserRepository,
	eventPublisher domain.EmailEventPublisher,
	assetSvc AssetStoragePort,
	logger *slog.Logger,
) *EmailService {
	return &EmailService{
		cfg:              cfg,
		delivery:         delivery,
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		eventPublisher:   eventPublisher,
		assetSvc:         assetSvc,
		logger:           logger,
	}
}

// SendGenericEmail sends an arbitrary email supplied by administrators.
func (s *EmailService) SendGenericEmail(ctx context.Context, payload *dto.SendEmailRequest) (string, error) {
	if err := payload.Validate(); err != nil {
		return "", domain.NewValidationError(err.Error())
	}

	msg, err := s.buildGenericMessage(payload)
	if err != nil {
		return "", err
	}

	return s.dispatch(ctx, msg)
}

// AvailableSenders returns the approved, provider-verified identities that an
// administrator may choose for a manual email.
func (s *EmailService) AvailableSenders() ([]dto.EmailSenderOption, error) {
	addresses, err := s.allowedFromAddresses()
	if err != nil {
		return nil, err
	}

	options := make([]dto.EmailSenderOption, 0, len(addresses))
	for _, address := range addresses {
		options = append(options, dto.EmailSenderOption{Name: address.Name, Address: address.Address})
	}
	return options, nil
}

// GetEmailHistory returns paginated email delivery records sourced from notification logs.
func (s *EmailService) GetEmailHistory(ctx context.Context, limit, offset int) ([]dto.EmailHistoryEntry, int64, error) {
	if s.notificationRepo == nil {
		return nil, 0, domain.NewInternalError(constants.MsgEmailHistoryRepoNotConfiguredVN, nil)
	}
	if limit <= 0 {
		return nil, 0, domain.NewValidationError(constants.MsgLimitMustBeGreaterThanZeroVN)
	}
	if offset < 0 {
		return nil, 0, domain.NewValidationError(constants.MsgOffsetCannotBeNegativeVN)
	}

	notifications, err := s.notificationRepo.ListByChannel(ctx, domain.NotificationChannelEmail, limit, offset)
	if err != nil {
		return nil, 0, domain.NewInternalError(constants.MsgFailedToRetrieveEmailHistoryVN, err)
	}

	total, err := s.notificationRepo.CountByChannel(ctx, domain.NotificationChannelEmail)
	if err != nil {
		return nil, 0, domain.NewInternalError(constants.MsgFailedToCountEmailHistoryVN, err)
	}

	senders := s.buildSenderLookup(ctx, notifications)

	history := make([]dto.EmailHistoryEntry, 0, len(notifications))
	for _, notification := range notifications {
		recipients := s.parseNotificationRecipients(notification.EmailRecipients)

		var messageID *string
		if notification.ResendMessageID != nil {
			trimmed := strings.TrimSpace(*notification.ResendMessageID)
			if trimmed != "" {
				messageIDCopy := trimmed
				messageID = &messageIDCopy
			}
		}

		senderName := ""
		senderUsername := ""
		if sender, ok := senders[notification.SenderID]; ok && sender != nil {
			senderName = sender.Fullname
			senderUsername = sender.Username
		} else if notification.SenderID == constants.SystemUserID {
			senderName = constants.DefaultEmailSenderName
			senderUsername = constants.DefaultEmailSenderAddress
		}

		entry := dto.EmailHistoryEntry{
			ID:             notification.ID,
			Subject:        notification.Title,
			Body:           notification.Message,
			MessageID:      messageID,
			Recipients:     recipients,
			SenderID:       notification.SenderID,
			SenderName:     senderName,
			SenderUsername: senderUsername,
			Type:           string(notification.Type),
			Channel:        string(notification.Channel),
			SentAt:         notification.CreatedAt,
		}

		history = append(history, entry)
	}

	return history, total, nil
}

func (s *EmailService) buildSenderLookup(ctx context.Context, notifications []*domain.Notification) map[uint]*domain.User {
	if len(notifications) == 0 {
		return map[uint]*domain.User{}
	}

	result := make(map[uint]*domain.User, len(notifications))
	for _, notification := range notifications {
		if notification == nil {
			continue
		}

		if notification.SenderID == 0 {
			continue
		}

		// Prefer the preloaded sender when available.
		if notification.Sender != nil {
			result[notification.SenderID] = notification.Sender
			continue
		}

		if s.userRepo == nil {
			continue
		}

		if _, exists := result[notification.SenderID]; exists {
			continue
		}

		user, err := s.userRepo.GetByID(ctx, notification.SenderID)
		if err != nil {
			if !domain.IsNotFoundError(err) && s.logger != nil {
				s.logger.Warn("failed to resolve notification sender", "error", err, "senderID", notification.SenderID)
			}
			continue
		}

		result[notification.SenderID] = user
	}

	return result
}

func (s *EmailService) buildGenericMessage(payload *dto.SendEmailRequest) (*domain.EmailMessage, error) {
	fromAddr, err := s.resolveFromAddress(payload.From)
	if err != nil {
		return nil, err
	}

	recipients, err := parseAddresses(payload.Recipients)
	if err != nil {
		return nil, err
	}

	cc, err := parseAddresses(payload.Cc)
	if err != nil {
		return nil, err
	}

	bcc, err := parseAddresses(payload.Bcc)
	if err != nil {
		return nil, err
	}

	var replyTo *domain.EmailAddress
	if payload.ReplyTo != "" {
		reply, err := domain.ParseEmailAddress(payload.ReplyTo)
		if err != nil {
			return nil, err
		}
		replyTo = &reply
	}

	textBody := payload.TextBody
	if textBody == "" {
		textBody = htmlToPlainText(payload.HTMLBody)
	}

	// Map DTO attachments to domain attachments
	attachments := make([]domain.EmailAttachment, 0, len(payload.Attachments))
	for _, att := range payload.Attachments {
		attachments = append(attachments, domain.EmailAttachment{
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Content:     att.Content,
		})
	}

	msg := &domain.EmailMessage{
		Kind:        domain.EmailKindGeneric,
		From:        fromAddr,
		ReplyTo:     replyTo,
		To:          recipients,
		CC:          cc,
		BCC:         bcc,
		Subject:     payload.Subject,
		HTMLBody:    payload.HTMLBody,
		TextBody:    textBody,
		Attachments: attachments,
	}

	if err := msg.Validate(); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *EmailService) dispatch(ctx context.Context, msg *domain.EmailMessage) (string, error) {
	timeout := s.cfg.SendTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	sendCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := s.delivery.Send(sendCtx, msg)
	if err != nil {
		s.logger.Error("email send failed", "error", err, "subject", msg.Subject)
		return "", domain.NewInternalError(constants.MsgEmailDeliveryFailedVN, err)
	}

	// Publish email sent event for audit logging (Observer pattern)
	if s.eventPublisher != nil {
		var initiatedBy *uint
		if userID := auditctx.GetUserID(ctx); userID != nil && *userID > 0 {
			initiatedBy = userID
		}

		event := domain.NewEmailSentEvent(result, msg, initiatedBy)

		if err := s.eventPublisher.PublishEmailSent(ctx, event); err != nil {
			s.logger.Error("failed to publish email sent event", "error", err, "message_id", result.MessageID)
		}
	}

	return result.MessageID, nil
}

func (s *EmailService) resolveFromAddress(raw string) (domain.EmailAddress, error) {
	allowed, err := s.allowedFromAddresses()
	if err != nil {
		return domain.EmailAddress{}, err
	}
	if strings.TrimSpace(raw) == "" {
		return allowed[0], nil
	}

	requested, err := domain.ParseEmailAddress(raw)
	if err != nil {
		return domain.EmailAddress{}, err
	}
	for _, address := range allowed {
		if address.Address == requested.Address {
			return address, nil
		}
	}

	return domain.EmailAddress{}, domain.NewValidationError("địa chỉ email gửi chưa được cấu hình hoặc xác minh")
}

func (s *EmailService) allowedFromAddresses() ([]domain.EmailAddress, error) {
	addresses := make([]domain.EmailAddress, 0, len(approvedEmailSenders))
	for _, raw := range approvedEmailSenders {
		address, parseErr := domain.ParseEmailAddress(raw)
		if parseErr != nil {
			return nil, domain.NewInternalError("địa chỉ email gửi cố định không hợp lệ", parseErr)
		}
		addresses = append(addresses, address)
	}

	return addresses, nil
}

func parseAddresses(values []string) ([]domain.EmailAddress, error) {
	addresses := make([]domain.EmailAddress, 0, len(values))
	for _, raw := range values {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		addr, err := domain.ParseEmailAddress(raw)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}
	return addresses, nil
}

func (s *EmailService) parseNotificationRecipients(raw *string) []dto.EmailRecipient {
	if raw == nil {
		return nil
	}

	payload := strings.TrimSpace(*raw)
	if payload == "" {
		return nil
	}

	var stored []domain.EmailAddress
	if err := json.Unmarshal([]byte(payload), &stored); err != nil {
		if s.logger != nil {
			s.logger.Warn("failed to parse stored email recipients", "error", err)
		}
		return nil
	}

	recipients := make([]dto.EmailRecipient, 0, len(stored))
	for _, recipient := range stored {
		recipients = append(recipients, dto.EmailRecipient{
			Name:    recipient.Name,
			Address: recipient.Address,
		})
	}

	return recipients
}

func htmlToPlainText(html string) string {
	if html == "" {
		return ""
	}
	replacer := strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n", "</p>", "\n", "</li>", "\n")
	plain := replacer.Replace(html)
	stripTags := regexp.MustCompile(`<[^>]+>`)
	plain = stripTags.ReplaceAllString(plain, "")
	plain = strings.ReplaceAll(plain, "&nbsp;", " ")
	plain = strings.TrimSpace(plain)
	return plain
}
