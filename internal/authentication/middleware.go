package authentication

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Implement authentication logic here
		fmt.Println("AuthMiddleware")
		return c.Next()
	}
}
