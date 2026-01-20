package controllers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type TransferToken struct {
	TransferID string `json:"transfer_id"`
	Token      string `json:"token"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

type ReqToGenTokenPDF struct {
	TransferId      string          `json:"transferId"`
}

type ReqShortLink struct {
	Link string `json:"link"`
}

type RespShortLink struct {
	Data struct {
		ShortLink string `json:"short-link"`
	} `json:"data"`
}

type SendSMTPReportRequest struct {
	AccountName   string          `json:"accountName"`
	MinDateTime  string          `json:"minDateTime"`
	MaxDateTime  string          `json:"maxDateTime"`
	SumTxnCount  string          `json:"sumTxnCount"`
	SumTxnAmount string          `json:"sumTxnAmount"`
	UrlDownload  string          `json:"urlDownload"`
	MailTo       string          `json:"mailTo"`
	RandomID     int             `json:"randomID"`
	Transfers     []TransferToken `json:"transfers"`

}



func buildDownloadLink(token string) string {
	baseURL := os.Getenv("BASE_DOWNLOAD_URL")
	return fmt.Sprintf("%s?token=%s", baseURL, url.QueryEscape(token))
}

func cleanEmails(emails string) []string {
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

func goToSMTP(
	accountName string,
	minDateTime string,
	maxDateTime string,
	sumTxnCount string,
	sumTxnAmount string,
	urlDownload string,
	mailTo string,
	randomID int,
) error {

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := os.Getenv("MAIL_FROM")
	fromName := os.Getenv("MAIL_FROM_NAME")

		// Step 1: Get short link

	reqShortLink := ReqShortLink{Link: urlDownload}

	jsonData, err := json.Marshal(reqShortLink)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		os.Getenv("URL_ONE_GENERATE_SHOT_LINK"),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var respShortLink RespShortLink
	if err := json.Unmarshal(body, &respShortLink); err != nil {
		return err
	}

	if respShortLink.Data.ShortLink == "" {
		return fmt.Errorf("short link empty")
	}

	link := respShortLink.Data.ShortLink
	log.Printf("[%d]Generated Short Link: %s", randomID, link)

		// Step 2: Prepare the email

	fromHeader := fmt.Sprintf("%s <%s>", fromName, smtpFrom)
	subject := "รายงานโอนเงินกลับประจำวัน"

	bodyText := fmt.Sprintf(`<p>เรียน %s,</p><br/>
		<p>ทางบริษัทฯ ส่วนงานการรับชำระเงิน ได้ส่งรายงานการโอนเงิน Online Payment Services (OPS) ประจำวันที่ %s มาให้ท่าน โดยมีรายละเอียดดังนี้</p><br/><br/>
		<p>จำนวนรายการ : %s รายการ</p>
		<p>ยอดรับชำระเงิน : %s บาท</p><br/>
		<p>ทั้งนี้ สามารถดาวน์โหลดรายละเอียดการรับเงินได้ที่ %s</p><br/>
		<p>หากท่านต้องการข้อมูลเพิ่มเติม โปรดติดต่อ ทางบริษัทฯ ส่วนงานการรับชำระเงิน ผ่านช่องทางต่าง ๆ ดังนี้</p>
		<p>E-mail : online-support@inet.co.th</p>
		<p>ขอแสดงความนับถือ</p>
		<p>บริษัทฯ ส่วนงานการรับชำระเงิน</p>`,
		accountName, time.Now().Format("02/01/2006"), sumTxnCount, sumTxnAmount, link)
	
	to := cleanEmails(mailTo)
	bcc := cleanEmails(os.Getenv("MAIL_BCC"))
	allRecipients := append(to, bcc...)

	msg := []byte(
		"Return-Path: " + smtpFrom + "\r\n" +
			"From: " + fromHeader + "\r\n" +
			"To: " + strings.Join(to, ",") + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Date: " + time.Now().Format(time.RFC1123Z) + "\r\n" +
			"Message-ID: <" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@thaidotcompayment.co.th>\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n\r\n" +
			bodyText,
	)

	// ===== Connect using STARTTLS =====
	conn, err := net.Dial("tcp", smtpHost+":"+smtpPort)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		return err
	}
	defer client.Quit()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: smtpHost}
		if err := client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}

	// ===== Enable SMTP Auth (สำคัญมากสำหรับ SPF/DMARC) =====
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	if err := client.Auth(auth); err != nil {
		return err
	}

	if err := client.Mail(smtpFrom); err != nil {
		return err
	}

	for _, addr := range allRecipients {
		if err := client.Rcpt(addr); err != nil {
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = w.Write(msg)
	return err
}


func GenPDFLinks(tokens []TransferToken) []string {
	var links []string

	base := os.Getenv("URL_LINKPDF")

	for _, t := range tokens {
		if t.Token == "" {
			continue
		}
		links = append(links, base+t.Token)
	}

	return links
}

func PostJSONGentokenPDF(transferId string) (string, error) {

	url := os.Getenv("URL_LINK_GEN_TOKEN_PDF")
	secret := os.Getenv("SECRECT_KEY")

	payload := map[string]interface{}{
		"key":  secret,
		"type": "Transfer",
		"details": []map[string]string{
			{
				"transfer_id": transferId,
			},
		},
	}

	jsonData, _ := json.Marshal(payload)

	log.Println("REQUEST TO GEN TOKEN:", string(jsonData))

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Println("RAW TOKEN RESPONSE:", string(body))

	var res []struct {
		TransferID string `json:"transfer_id"`
		Token      string `json:"token"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return "", err
	}

	if len(res) == 0 || res[0].Token == "" {
		return "", fmt.Errorf("token empty")
	}

	return res[0].Token, nil
}

func SendSMTPReport(c *fiber.Ctx) error {

	var tokens []TransferToken

	if err := c.BodyParser(&tokens); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"responseCode": "99",
			"responseMessage": "invalid request body",
		})
	}

	if len(tokens) == 0 {
		return c.Status(400).JSON(fiber.Map{
			"responseCode": "98",
			"responseMessage": "token list empty",
		})
	}

	basePDF := os.Getenv("URL_LINKPDF")
	accountName  := "Test Name"
	minDateTime  := "2026-01-01"
	maxDateTime  := "2026-02-01"
	sumTxnCount  := "100"
	sumTxnAmount := "10000"
	mailTo       := "boxblue779@email.com"
	randomID     := 1000

	for i, t := range tokens {

		if t.Token == "" {
			log.Printf("[%d] token empty, skip transfer_id=%s", i, t.TransferID)
			continue
		}

		longLink := basePDF + t.Token
		log.Printf("[%d] transfer_id=%s pdf=%s", i, t.TransferID, longLink)

		err := goToSMTP(
			accountName,
			minDateTime,
			maxDateTime,
			sumTxnCount,
			sumTxnAmount,
			longLink,     
			mailTo,
			randomID+i,
		)

		if err != nil {
			log.Printf("[%d] send mail failed (%s): %v", i, t.TransferID, err)
		}
	}

	return c.Status(200).JSON(fiber.Map{
		"responseCode": "00",
		"responseMessage": "Success",
	})
}






