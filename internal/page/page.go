package page

type Params struct {
	Limit  int
	Offset int
	Sort   string
}

type Metadata struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type Result[T any] struct {
	Items      []T
	Pagination Metadata
}

func NewResult[T any](items []T, params Params, total int) Result[T] {
	return Result[T]{
		Items: items,
		Pagination: Metadata{
			Limit:  params.Limit,
			Offset: params.Offset,
			Total:  total,
		},
	}
}
