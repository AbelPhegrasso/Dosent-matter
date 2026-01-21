package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

var validate = validator.New()

type DetailRes struct {
	TransferId  string `json:"transfer_id"`
	RecipientId string `json:"recipient_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
}

type ReceiveResfomat struct {
	Key    string      `json:"key" validate:"required"`
	Type   string      `json:"type" validate:"required,oneof=Transfer Income"`
	Detail []DetailRes `json:"details" validate:"min=1,dive"`
}

type SentNext struct {
	TranfersIdSentOut string `json:"transferId" validate:"required"`
}
type TokenWithId struct {
	TransferId string `json:"transfer_id"`
	Token      string `json:"token"`
}
type APIResponseToUsers struct {
	TransferId string `json:"transfer_id"`
	Token      string `json:"token"`
	Fullurl    string `json:"full_url"`
	Shoturl    string `json:"short_url"`
}

func HandleAPI(c *fiber.Ctx) error {
	var req ReceiveResfomat

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Cannot parse JSON",
			"detail": err.Error(),
		})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Validation failed",
			"detail": err.Error(),
		})
	}
	systemKey := os.Getenv("SECRECT_KEY")
	if req.Key != systemKey {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Key",
		})
	}

	if req.Type != "Transfer" && req.Type != "transfer" && req.Type != "Income" && req.Type != "income" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Type should be Income or Transfer ",
		})
	}
	resultUrl, err := GenToken(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

}

func GenToken(req ReceiveResfomat) (string, error) {
	log.Printf("Received Type: %s with %d details", req.Type, len(req.Detail))
	var IdResult []SentNext

	for _, v := range req.Detail {
		re := SentNext{
			TranfersIdSentOut: v.TransferId,
		}
		IdResult = append(IdResult, re) // output jaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
	}

	type APIResponse struct {
		Token string `json:"token"`
	}

	var tkid []TokenWithId
	for i, v := range IdResult {
		payload, _ := json.Marshal(map[string]string{"transferId": v.TranfersIdSentOut})
		resp, err := http.Post(os.Getenv("URL_ONE_GENERATE_TOKEN"), "application/json", bytes.NewBuffer(payload))
		if err != nil {
			log.Printf("Error sending request for item %d: %v", i, err)
		}
		func() {
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				log.Printf("Request failed status: %d", resp.StatusCode)
				return
			}

			var apiRes APIResponse
			if err := json.NewDecoder(resp.Body).Decode(&apiRes); err != nil {
				log.Printf("Error decoding response: %v", err)
				return
			}

			newItem := TokenWithId{
				TransferId: v.TranfersIdSentOut,
				Token:      apiRes.Token,
			}

			tkid = append(tkid, newItem)
		}()
	}
	urlResultList, err := UrlCreate(tkid)
	if err != nil {
		return "", err
	}

	if len(urlResultList) > 0 {
		return urlResultList[0].Shoturl, nil
	}

	return "", fmt.Errorf("no url generated")
}

////////////////////////////////////////////////////////////////////////

func UrlCreate(tkid []TokenWithId) ([]APIResponseToUsers, error) {
	var res []APIResponseToUsers
	for _, token := range tkid {
		fullUrl := os.Getenv("URL_LINK_FOLLOW_TOKEN") + token.Token

		var shortUrl string = ""
		maxRetries := 3

		for attempt := 1; attempt <= maxRetries; attempt++ {

			reqBody := bytes.NewBuffer([]byte(fullUrl))

			resp, err := http.Post(
				os.Getenv("URL_ONE_GENERATE_SHOT_LINK"),
				"application/json",
				reqBody,
			)

			if err == nil {
				defer resp.Body.Close()

				bodyBytes, _ := io.ReadAll(resp.Body)
				shortUrl = string(bodyBytes)
				break
			}

			log.Printf("Attempt %d/%d failed for token %s: %v", attempt, maxRetries, token.Token, err)

			if attempt < maxRetries {
				time.Sleep(1 * time.Second)
			}
		}
		r := APIResponseToUsers{
			TransferId: token.TransferId,
			Token:      token.Token,
			Fullurl:    fullUrl,
			Shoturl:    shortUrl,
		}
		res = append(res, r)
	}

	return res, nil
}
