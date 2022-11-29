package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

var (
	// MaxPaginationSize represents the maximum number of records that can be returned per page
	MaxPaginationSize = 1000
	// DefaultPaginationSize represents the default number of records that are returned per page
	DefaultPaginationSize = 100
	// context keys for storing pagination information
	ctxKeyLimit  = "pager_limit"
	ctxKeyPage   = "pager_page"
	ctxKeyOrder  = "pager_order"
	ctxKeyParsed = "pager_parsed"
)

func ParsePagination(c *gin.Context) {
	// Initializing default
	limit := DefaultPaginationSize
	page := 1
	order := ""
	query := c.Request.URL.Query()

	for key, value := range query {
		queryValue := value[len(value)-1]

		switch key {
		case "limit":
			limit, _ = strconv.Atoi(queryValue)
		case "page":
			page, _ = strconv.Atoi(queryValue)
		case "order":
			order = queryValue
		}
	}

	switch {
	case limit > MaxPaginationSize:
		limit = MaxPaginationSize
	case limit <= 0:
		limit = DefaultPaginationSize
	}

	c.Set(ctxKeyLimit, limit)
	c.Set(ctxKeyPage, page)
	c.Set(ctxKeyOrder, order)
	c.Set(ctxKeyParsed, true)
}

// RequestQueryMods parses a gin request and returns the appropriate query mods
func RequestQueryMods(c *gin.Context) []qm.QueryMod {
	limit, page := getLimitAndPage(c)
	mods := []qm.QueryMod{qm.Limit(limit)}

	if o := offset(page, limit); o != 0 {
		mods = append(mods, qm.Offset(o))
	}

	return mods
}

func GetOrder(c *gin.Context) string {
	if !c.GetBool(ctxKeyParsed) {
		ParsePagination(c)
	}

	return c.GetString(ctxKeyOrder)
}

func SetHeaders(c *gin.Context, count int) {
	limit, page := getLimitAndPage(c)

	c.Header("Pagination-Count", strconv.Itoa(count))
	c.Header("Pagination-Limit", strconv.Itoa(limit))
	c.Header("Pagination-Page", strconv.Itoa(page))
}

func offset(page, limit int) int {
	if page == 0 {
		return 0
	}

	return (page - 1) * limit
}

func getLimitAndPage(c *gin.Context) (int, int) {
	if !c.GetBool(ctxKeyParsed) {
		ParsePagination(c)
	}

	return c.GetInt(ctxKeyLimit), c.GetInt(ctxKeyPage)
}
