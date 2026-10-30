package racing

// PointsForPosition returns the fixed championship points awarded for a
// finishing position. Only P1..P6 score: 9, 6, 4, 3, 2, 1.
func PointsForPosition(position int) int {
	switch position {
	case 1:
		return 9
	case 2:
		return 6
	case 3:
		return 4
	case 4:
		return 3
	case 5:
		return 2
	case 6:
		return 1
	default:
		return 0
	}
}
