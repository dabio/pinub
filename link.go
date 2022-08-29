package pinub

import (
	"context"
	"database/sql"
	"time"
)

type Link struct {
	ID        string
	URL       string
	CreatedAt *time.Time
}

type LinkService struct {
	DB *sql.DB
}

func (service *LinkService) Links(ctx context.Context, uid string) ([]Link, error) {
	query := `
		SELECT id, url, ul.created_at FROM links AS l
			JOIN user_links AS ul ON l.id = ul.link_id AND ul.user_id = $1
		ORDER BY ul.created_at DESC`

	rows, err := service.DB.QueryContext(ctx, query, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var link Link
		if err = rows.Scan(&link.ID, &link.URL, &link.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}
