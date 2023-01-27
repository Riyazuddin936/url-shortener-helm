package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/riyazuddin/shortener/database"
	"github.com/riyazuddin/shortener/helpers"
)

type request struct {
	URL    string        `json:"url"`
	Expiry time.Duration `json:"expiry"`
}

type response struct {
	URL             string        `json:"url"`
	Short           string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaining  int           `json:"rate_limit"`
	XRateLimitReset time.Duration `json:"rate_limit_reset"`
}

// ShortenURL ...
func ShortenURL(c *fiber.Ctx) error {

	body := new(request)
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot parse JSON",
		})
	}
	r1 := database.CreateClient(1)
	defer r1.Close()

	_, isIpPresent := r1.Get(database.Ctx, c.IP()).Result()
	if isIpPresent == redis.Nil {
		r1.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		val, _ := r1.Get(database.Ctx, c.IP()).Result()
		isQuotaUp, _ := strconv.Atoi(val)
		if isQuotaUp <= 0 {
			limit, _ := r1.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":            "Rate limit exceeded",
				"rate_limit_reset": limit / time.Nanosecond / time.Minute,
			})
		}
	}

	body.URL = helpers.EnforceHTTP(body.URL)

	id := uuid.New().String()[:6]

	r0 := database.CreateClient(0)
	defer r0.Close()

	_, isIdUsed := r0.Get(database.Ctx, id).Result()

	if isIdUsed == redis.Nil {
		r0.Set(database.Ctx, id, body.URL, body.Expiry*3600*time.Second).Err()

	} else {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Id is not unique",
		})
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	resp := response{
		URL:             body.URL,
		Short:           "localhost:4030/" + id,
		Expiry:          body.Expiry,
		XRateRemaining:  10,
		XRateLimitReset: 30,
	}
	r1.Decr(database.Ctx, c.IP())
	val, _ := r1.Get(database.Ctx, c.IP()).Result()
	resp.XRateRemaining, _ = strconv.Atoi(val)
	ttl, _ := r1.TTL(database.Ctx, c.IP()).Result()
	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute

	return c.Status(fiber.StatusOK).JSON(resp)

}
