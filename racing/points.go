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

// PodiumForPosition returns the podium counters to award for a finishing
// position: P1 counts as a win (and gold), P2 as silver, P3 as bronze.
func PodiumForPosition(position int) (wins, gold, silver, bronze int) {
	switch position {
	case 1:
		return 1, 1, 0, 0
	case 2:
		return 0, 0, 1, 0
	case 3:
		return 0, 0, 0, 1
	}
	return 0, 0, 0, 0
}
