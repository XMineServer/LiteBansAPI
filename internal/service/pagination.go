package service

import "xmine/litebans-api/internal/domain"

// ResolveOffsetPage validates/clamps page & pageSize for offset-based list endpoints (4.1 TOR).
func ResolveOffsetPage(pageParam, pageSizeParam *int, defaultPageSize, maxPageSize int) (page, pageSize int, err error) {
	page = 1
	if pageParam != nil {
		if *pageParam < 1 {
			return 0, 0, domain.NewInvalidParameter("page must be >= 1")
		}
		page = *pageParam
	}
	pageSize = defaultPageSize
	if pageSizeParam != nil {
		if *pageSizeParam < 1 {
			return 0, 0, domain.NewInvalidParameter("pageSize must be >= 1")
		}
		pageSize = *pageSizeParam
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize, nil
}

// TotalPages computes the number of pages for a given total item count and page size.
func TotalPages(totalItems int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	pages := int(totalItems / int64(pageSize))
	if totalItems%int64(pageSize) != 0 {
		pages++
	}
	return pages
}
