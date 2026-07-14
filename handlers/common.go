package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// errStatus maps an error (matched via errors.Is) to an HTTP status code.
type errStatus struct {
	Err    error
	Status int
}

// writeMappedError writes a JSON error response by matching err against the
// provided mappings. The first match wins. If nothing matches, 500 is returned.
func writeMappedError(c *gin.Context, err error, mappings []errStatus) {
	for _, m := range mappings {
		if errors.Is(err, m.Err) {
			c.JSON(m.Status, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

// parsePagination extracts page and pageSize from query parameters.
// Defaults are page=1, pageSize=20.
func parsePagination(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20
	if v, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(c.DefaultQuery("pageSize", "20")); err == nil && v > 0 {
		pageSize = v
	}
	return page, pageSize
}

// paginatedResponse writes a JSON response with items, total, page and pageSize.
func paginatedResponse(c *gin.Context, items any, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
