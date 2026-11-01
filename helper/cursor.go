package helper

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"latihan-fiber/app/model"
)

var ErrInvalidCursor = BadRequest("kursor tidak valid")

func EncodeCursor(createdAt time.Time, id int) string {
	raw := strconv.FormatInt(createdAt.UTC().UnixNano(), 10) +
		"|" + strconv.Itoa(id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(encoded string) (model.Cursor, error) {
	if strings.TrimSpace(encoded) == "" {
		return model.Cursor{}, ErrInvalidCursor
	}

	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return model.Cursor{}, ErrInvalidCursor
	}

	parts := strings.Split(string(decoded), "|")
	if len(parts) != 2 {
		return model.Cursor{}, ErrInvalidCursor
	}

	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return model.Cursor{}, ErrInvalidCursor
	}

	id, err := strconv.Atoi(parts[1])
	if err != nil || id < 1 {
		return model.Cursor{}, ErrInvalidCursor
	}

	return model.Cursor{CreatedAt: time.Unix(0, nanos).UTC(), ID: id}, nil
}

func ParseCursorQuery(c *fiber.Ctx) (model.CursorQuery, error) {
	limit := c.QueryInt("limit", 10)
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	search := strings.TrimSpace(c.Query("search"))

	var isActive *bool
	if val := c.Query("is_active"); val != "" {
		parsed, err := strconv.ParseBool(val)
		if err != nil {
			return model.CursorQuery{}, BadRequest("parameter is_active harus bernilai boolean (true/false)")
		}
		isActive = &parsed
	}

	var cursorPtr *model.Cursor
	if encodedCursor := c.Query("cursor"); encodedCursor != "" {
		cursor, err := DecodeCursor(encodedCursor)
		if err != nil {
			return model.CursorQuery{}, err
		}
		cursorPtr = &cursor
	}

	return model.CursorQuery{
		Limit:    limit,
		Search:   search,
		IsActive: isActive,
		After:    cursorPtr,
	}, nil
}
