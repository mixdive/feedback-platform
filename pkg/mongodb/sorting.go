package mongodb

type SortDirection string //@name SortDirection

const (
	SortAsc  SortDirection = "asc"  //@name Ascending
	SortDesc SortDirection = "desc" //@name Descending
)

func (sd SortDirection) Value() int {
	switch sd {
	case "asc":
		return 1
	case "desc":
		return -1
	default:
		return 1
	}
}
