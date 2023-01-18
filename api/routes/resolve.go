package routes

import (
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/riyazuddin/shortener/database"
)

// ResolveURL ...
func ResolveURL(c *fiber.Ctx) error {
	// get the short from the url
	url := c.Params("url")

	r0 := database.CreateClient(0)
	defer r0.Close()
	value, err := r0.Get(database.Ctx, url).Result()
	if err == redis.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "short not found on database",
		})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot connect to DB",
		})
	}

	return c.Redirect(value, 301)
}
