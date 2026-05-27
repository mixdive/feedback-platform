package mongodb

type Paging struct {
	PageNumber int `form:"pageNumber" json:"pageNumber" default:"1"` // PageNumber defines the page number to retrieve.
	PageSize   int `form:"pageSize" json:"pageSize" default:"100"`   // PageSize defines the number of items to include per page with a default value of 100.
} //@name Paging

func NewPaging(pageNumber int, pageSize int) *Paging {
	return &Paging{
		PageNumber: pageNumber,
		PageSize:   pageSize,
	}
}

func (p *Paging) SkipValue() int64 {
	pageNumber := int64(p.PageNumber)

	if pageNumber < 1 {
		pageNumber = 1
	}

	skip := (pageNumber - 1) * p.PageSizeValue(100)
	return skip
}

func (p *Paging) PageSizeValue(defaultPageSize int) int64 {
	pageSize := int64(p.PageSize)
	if pageSize <= 0 {
		pageSize = int64(defaultPageSize)
	}

	return pageSize
}
