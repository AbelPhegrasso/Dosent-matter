package controllers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"

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

type ReceiveResFormat struct {
	Key    string      `json:"key" validate:"required"`
	Type   string      `json:"type" validate:"required,oneof=Transfer Income"`
	Detail []DetailRes `json:"details" validate:"min=1,dive"`
}

type SentNext struct {
	TransferIdSentOut string `json:"transferId" validate:"required"`
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

type MailDetail struct {
	AccountName  string `json:"accountName"`
	MinDateTime  string `json:"minDateTime"`
	MaxDateTime  string `json:"maxDateTime"`
	SumTxnCount  string `json:"sumTxnCount"`
	SumTxnAmount string `json:"sumTxnAmount"`
}

type MailPayload struct {
	FromHeader string
	Subject    string
	Body       string
	To         []string
	Bcc        []string
	ShortLink  string
	FullLink   string
	TransferId string
}

type MailSendResult struct {
	TransferId string `json:"transfer_id"`
	Email      string `json:"receiver_email"`
	ShortLink  string `json:"short_link"`
	FullLink   string `json:"full_link"`
	Status     string `json:"status"` // SUCCESS | FAIL
	Error      string `json:"error,omitempty"`
}

func HandleAPI(c *fiber.Ctx) error {
	var req ReceiveResFormat

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

	urlResults, err := GenToken(req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	mailResults := processMailSend(urlResults)

	success := 0
	fail := 0
	for _, r := range mailResults {
		if r.Status == "SUCCESS" {
			success++
		} else {
			fail++
		}
	}

	return c.Status(200).JSON(fiber.Map{
		"responseCode": "00",
		"summary": fiber.Map{
			"total":   len(mailResults),
			"success": success,
			"fail":    fail,
		},
		"results": mailResults,
	})
}

func GenToken(req ReceiveResFormat) ([]APIResponseToUsers, error) {
	log.Printf("Received Type: %s with %d details", req.Type, len(req.Detail))
	var IdResult []SentNext

	for _, v := range req.Detail {
		re := SentNext{
			TransferIdSentOut: v.TransferId,
		}
		IdResult = append(IdResult, re) // output jaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
	}

	type APIResponse struct {
		Token string `json:"token"`
	}

	var tkid []TokenWithId
	for i, v := range IdResult {
		payload, _ := json.Marshal(map[string]string{"transferId": v.TransferIdSentOut})
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
				TransferId: v.TransferIdSentOut,
				Token:      apiRes.Token,
			}

			tkid = append(tkid, newItem)
		}()
	}

	urlResultList, err := UrlCreate(tkid)
	if err != nil {
		return nil, err
	}

	return urlResultList, nil
}

////////////////////////////////////////////////////////////////////////

func UrlCreate(tkid []TokenWithId) ([]APIResponseToUsers, error) {
	var res []APIResponseToUsers
	for _, token := range tkid {
		fullUrl := os.Getenv("URL_LINK_FOLLOW_TOKEN") + token.Token

		var shortUrl string = ""
		maxRetries := 3

		reqObj := map[string]string{
			"link": fullUrl,
		}
		jsonBody, err := json.Marshal(reqObj)
		if err != nil {
			return nil, err
		}

		for attempt := 1; attempt <= maxRetries; attempt++ {

			resp, err := http.Post(
				os.Getenv("URL_ONE_GENERATE_SHOT_LINK"),
				"application/json",
				bytes.NewBuffer(jsonBody),
			)
			if resp.StatusCode != http.StatusOK {
				log.Printf("Shortlink's Attempt %d/%d failed (HTTP Status %s)", attempt, maxRetries, resp.Status)
				shortUrl = ""
				continue
			}
			if err == nil {
				defer resp.Body.Close()

				bodyBytes, _ := io.ReadAll(resp.Body)

				var slResp struct {
					Data struct {
						ShortLink string `json:"short-link"`
					} `json:"data"`
				}

				if err := json.Unmarshal(bodyBytes, &slResp); err == nil {
					shortUrl = slResp.Data.ShortLink
				}
				break
			}

			if attempt < maxRetries {
				time.Sleep(3 * time.Second)
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

func cleanEmail(emails string) []string {
	raw := strings.Split(emails, ",")
	var result []string
	for _, e := range raw {
		e = strings.TrimSpace(e) // ตัดช่องว่างซ้าย/ขวา
		if e != "" {
			result = append(result, e)
		}
	}
	return result
}

func processMailSend(urlResults []APIResponseToUsers) []MailSendResult {

	var results []MailSendResult

	for _, r := range urlResults {

		var mail MailDetail
		payload := gotoMail(&mail, r)

		err := sendMail(payload)

		result := MailSendResult{
			TransferId: r.TransferId,
			Email:      strings.Join(payload.To, ","),
			ShortLink:  payload.ShortLink,
			FullLink:   payload.FullLink,
		}

		if err != nil {
			result.Status = "FAIL"
			result.Error = err.Error()
		} else {
			result.Status = "SUCCESS"
		}

		results = append(results, result)
	}

	return results
}

func gotoMail(m *MailDetail, res APIResponseToUsers) MailPayload {

	m.AccountName = "Name"
	m.MinDateTime = "2026-01-01"
	m.MaxDateTime = "2026-01-02"
	m.SumTxnCount = "100"
	m.SumTxnAmount = "1000"

	link := res.Shoturl
	if link == "" {
		link = "not available"
	}

	fromName := os.Getenv("MAIL_FROM_NAME")
	smtpFrom := os.Getenv("MAIL_FROM")

	body := fmt.Sprintf(
		`<p>เรียน %s,</p><br/>
        <p>ทางบริษัทฯ ส่วนงานการรับชำระเงิน ได้ส่งรายงานการโอนเงิน Online Payment Services (OPS) ประจำวันที่ %s มาให้ท่าน โดยมีรายละเอียดดังนี้</p><br/><br/>
        <p>จำนวนรายการ : %s รายการ</p>
        <p>ยอดรับชำระเงิน : %s บาท</p><br/>
        <p>ทั้งนี้ สามารถดาวน์โหลดรายละเอียดการรับเงินได้ที่ %s</p><br/>
        <p>หากท่านต้องการข้อมูลเพิ่มเติม โปรดติดต่อ ทางบริษัทฯ ส่วนงานการรับชำระเงิน ผ่านช่องทางต่าง ๆ ดังนี้</p>
        <p>E-mail : online-support@inet.co.th</p>
        <p>ขอแสดงความนับถือ</p>
        <p>บริษัทฯ ส่วนงานการรับชำระเงิน</p>`,
		m.AccountName, time.Now().Format("02/01/2006"), m.SumTxnCount, m.SumTxnAmount, link)

	return MailPayload{
		FromHeader: fmt.Sprintf("%s <%s>", fromName, smtpFrom),
		Subject:    "รายงานโอนเงินกลับประจำวัน",
		Body:       body,
		To:         cleanEmails("boxblue779@gmail.com"),
		Bcc:        cleanEmails(os.Getenv("MAIL_BCC")),
		ShortLink:  link,
		FullLink:   res.Fullurl,
		TransferId: res.TransferId,
	}

}

func sendMail(p MailPayload) error {

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := os.Getenv("MAIL_FROM")

	allRecipients := append(p.To, p.Bcc...)

	msg := []byte(
		"Return-Path: " + smtpFrom + "\r\n" +
			"From: " + p.FromHeader + "\r\n" +
			"To: " + strings.Join(p.To, ",") + "\r\n" +
			"Subject: " + p.Subject + "\r\n" +
			"Date: " + time.Now().Format(time.RFC1123Z) + "\r\n" +
			"Message-ID: <" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@thaidotcompayment.co.th>\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n\r\n" +
			p.Body,
	)

	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {

		conn, err := net.Dial("tcp", smtpHost+":"+smtpPort)
		if err != nil {
			lastErr = err
			log.Printf("%s [SMTP] attempt %d/%d dial error: %v", p.TransferId, attempt, maxRetries, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		client, err := smtp.NewClient(conn, smtpHost)
		if err != nil {
			conn.Close()
			lastErr = err
			log.Printf("%s [SMTP] attempt %d/%d client error: %v", p.TransferId, attempt, maxRetries, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: smtpHost}); err != nil {
				client.Quit()
				conn.Close()
				lastErr = err
				log.Printf("%s [SMTP] attempt %d/%d starttls error: %v", p.TransferId, attempt, maxRetries, err)
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
		}

		if err := client.Auth(smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)); err != nil {
			client.Quit()
			conn.Close()
			lastErr = err
			log.Printf("%s [SMTP] attempt %d/%d auth error: %v", p.TransferId, attempt, maxRetries, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if err := client.Mail(smtpFrom); err != nil {
			client.Quit()
			conn.Close()
			lastErr = err
			log.Printf("%s [SMTP] attempt %d/%d mail from error: %v", p.TransferId, attempt, maxRetries, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		for _, addr := range allRecipients {
			if err := client.Rcpt(addr); err != nil {
				client.Quit()
				conn.Close()
				lastErr = err
				log.Printf("%s [SMTP] attempt %d/%d rcpt error: %v", p.TransferId, attempt, maxRetries, err)
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
		}

		w, err := client.Data()
		if err != nil {
			client.Quit()
			conn.Close()
			lastErr = err
			log.Printf("%s [SMTP] attempt %d/%d data error: %v", p.TransferId, attempt, maxRetries, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if _, err := w.Write(msg); err != nil {
			w.Close()
			client.Quit()
			conn.Close()
			lastErr = err
			log.Printf("%s [SMTP] attempt %d/%d write error: %v", p.TransferId, attempt, maxRetries, err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		w.Close()
		client.Quit()
		conn.Close()

		log.Printf("%s [SMTP] send success on attempt %d", p.TransferId, attempt)
		return nil
	}

	return fmt.Errorf("smtp failed after %d retries: %w", maxRetries, lastErr)
}
