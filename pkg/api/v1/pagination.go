package api

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

var (
	// maxPaginationSize represents the maximum number of records that can be returned per page
	maxPaginationSize = 1000

	// defaultPaginationSize represents the default number of records that are returned per page
	defaultPaginationSize = 100
)

// PaginationParams allow you to paginate the results
type PaginationParams struct {
	Limit   int    `json:"limit,omitempty"`
	Page    int    `json:"page,omitempty"`
	Cursor  string `json:"cursor,omitempty"`
	Preload bool   `json:"preload,omitempty"`
	OrderBy string `json:"orderby,omitempty"`
}

func parsePagination(c echo.Context) PaginationParams {
	var (
		limit = defaultPaginationSize
		page  = 1
		query = c.Request().URL.Query()
	)

	for key, value := range query {
		queryValue := value[len(value)-1]

		switch key {
		case "limit":
			limit, _ = strconv.Atoi(queryValue)
		case "page":
			page, _ = strconv.Atoi(queryValue)
		}
	}

	return PaginationParams{
		Limit: limit,
		Page:  page,
	}
}

func (p *PaginationParams) limitUsed() int {
	var limit int

	switch {
	case p.Limit > maxPaginationSize:
		limit = maxPaginationSize
	case p.Limit <= 0:
		limit = defaultPaginationSize
	default:
		limit = int(p.Limit)
	}

	return limit
}

func (p *PaginationParams) page() int {
	if p.Page < 1 {
		return 1
	}

	return p.Page
}

func (p *PaginationParams) offset() int {
	return (p.page() - 1) * p.limitUsed()
}

func (p *PaginationParams) getPageOffset() int {
	return p.limitUsed() * (p.page() - 1)
}

func (p *PaginationParams) queryMods() []qm.QueryMod {
	var mods []qm.QueryMod

	mods = append(mods, qm.Limit(p.limitUsed()))

	if p.Page > 0 {
		mods = append(mods, qm.Offset(p.offset()))
	}

	return mods
}
