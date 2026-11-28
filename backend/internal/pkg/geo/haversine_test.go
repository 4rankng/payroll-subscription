package geo

// 1° of longitude at the equator = R * π/180 = 111194.93m for R=6371000.
// Pins both the earth-radius constant and the degree→radian conversion so a
// future edit (e.g. swapping to km / wrong constant) is caught.

// Regression pin for the geofence check-in bypass: a point ~800m due south of
// LGD "Cổng A" must read ~800m so the 100m geofence can reject an off-site
// worker. 800m / 111195 m/deg ≈ 0.0071944° latitude.
