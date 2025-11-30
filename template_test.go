package i18n

import (
	"fmt"
	"testing"
	"time"
)

type Order struct {
	Count int
}

func TestParsePlaceholder(t *testing.T) {
	t.Run("ParsePlaceholder_Success", func(t *testing.T) {
		_, err := parsePlaceholder("{order.price|number:2}")
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("ParsePlaceholder_Fail", func(t *testing.T) {
		node, err := parsePlaceholder(" {count | eq:0?No items:{count} items}")
		if err != nil {
			t.Fatal(err)
		}
		if node == nil {
			t.Fatal("node is nil")
		}
		eval, err := node.Eval(map[string]any{
			"{count": Order{
				Count: 10,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if eval != "" {
			t.Fatal("eval should be empty")
		}
	})
}

func TestRenderTemplate(t *testing.T) {
	t.Run("RenderTemplate_Success", func(t *testing.T) {
		result, err := RenderTemplate(
			"User {user.hi|upper} {user.name | title}, price {order.price | number:2 | currency:¥}, items {count | eq:0?No items:{count} items}, created {order.created_at | date:2006/01/02}",
			map[string]any{
				"user": map[string]any{
					"name": "hello , 咸鱼",
					"hi":   "hi",
				},
				"order": map[string]any{
					"price":      12345.678,
					"created_at": time.Now(),
				},
				"count": 5,
			},
		)

		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(result)
	})

	t.Run("RenderTemplate_Struct", func(t *testing.T) {
		type User struct {
			Detail struct {
				Name     string
				Height   string
				Birthday string
			}
			Count float64
		}
		result, err := RenderTemplate(
			"{user.detail.name|title}, {user.count|eq:0?No items:{user.count} items} {user.detail.height|number:2} {user.detail.birthday|date:2006-01-02}",
			map[string]any{
				"user": &User{
					Detail: struct {
						Name     string
						Height   string
						Birthday string
					}{
						Name:     "咸鱼不翻身",
						Height:   "180",
						Birthday: "1990-01-01T15:11:20+08:00",
					},
					Count: 0,
				},
			})
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(result)
	})

}
