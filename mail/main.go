package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	gomail "github.com/ory/mail"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
)

type Mail struct {
	ID          uint
	Headers     map[string][]string
	Body        []uint
	Attachments []uint
	Embeds      []uint
}

type Content struct {
	ID          uint
	MailID      uint
	Data        []byte
	ContentType string
	Encoding    string
	Type        string
	Layout      string
	Name        string
}

type MailBox struct {
	Mails    map[uint]*Mail
	Contents map[uint]*Content
}

func main() {

	mailBox := &MailBox{}
	mailBox.Mails = make(map[uint]*Mail)
	mailBox.Contents = make(map[uint]*Content)

	buffer := &bytes.Buffer{}
	m := gomail.NewMessage()
	m.SetHeader("From", "from@example.com")
	m.SetHeader("To", "to@example.com")
	m.SetBody("text/plain", "வணக்கம்!")
	m.AddAlternative("text/html", "¡<b>Hola</b>, <i>señor</i>!</h1>")
	m.Attach("resources/test-attachment.pdf")
	m.Attach("resources/test-inline.png")
	m.Embed("resources/test-inline.png")
	m.WriteTo(buffer)

	msg, err := mail.ReadMessage(buffer)
	if err != nil {
		log.Fatal(err)
	}

	newMail := &Mail{}
	newMail.ID = uint(len(mailBox.Mails) + 1)
	newMail.Headers = msg.Header

	ProcessMailBody(msg.Body, msg.Header, mailBox, newMail, false, false, false)
	fmt.Printf("MailBox %v", mailBox)
}

func ProcessMailBody(body io.Reader, headers mail.Header, mailBox *MailBox, mail2 *Mail, isAttachment bool, isEmbeded bool, isAlt bool) error {
	mediaType, params, err := mime.ParseMediaType(headers.Get("Content-Type"))
	if err != nil {
		log.Fatal(err)
	}
	switch mediaType {
	case "multipart/alternative":
		isAlt = true
		mr := multipart.NewReader(body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				log.Fatal(err)
			}
			ProcessMailBody(part, mail.Header(part.Header), mailBox, mail2, isAttachment, isEmbeded, isAlt)
		}

	case "multipart/related":
		isEmbeded = true
		mr := multipart.NewReader(body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				log.Fatal(err)
			}
			ProcessMailBody(part, mail.Header(part.Header), mailBox, mail2, isAttachment, isEmbeded, isAlt)
		}

	case "multipart/mixed":
		isAttachment = true
		mr := multipart.NewReader(body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				log.Fatal(err)
			}
			ProcessMailBody(part, mail.Header(part.Header), mailBox, mail2, isAttachment, isEmbeded, isAlt)
		}
	case "text/plain":
		fallthrough
	case "text/html":
		fallthrough
	default:
		altContent := &Content{}
		altContent.ID = uint(len(mailBox.Contents) + 1)
		altContent.MailID = mail2.ID
		altContent.ContentType = headers.Get("Content-Type")
		altContent.Encoding = headers.Get("Content-Transfer-Encoding")
		mailBuffer := &bytes.Buffer{}
		switch strings.ToUpper(altContent.Encoding) {
		case "BASE64":
			_, err := mailBuffer.ReadFrom(base64.NewDecoder(base64.StdEncoding, body))
			if err != nil {
				return err
			}
		case "QUOTED-PRINTABLE":
			_, err := mailBuffer.ReadFrom(quotedprintable.NewReader(body))
			if err != nil {
				return err
			}
		case "8BIT", "7Bit":
			fallthrough
		default:
			_, err := mailBuffer.ReadFrom(body)
			if err != nil {
				return err
			}
		}
		altContent.Data = mailBuffer.Bytes()
		if isAlt {
			altContent.Type = "Alt"
			mail2.Body = append(mail2.Body, altContent.ID)
		} else if isEmbeded {
			altContent.Type = "Emb"
			altContent.Name = strings.TrimRight(strings.TrimLeft(headers.Get("Content-ID"), "<"), ">")
			altContent.Layout = strings.Split(headers.Get("Content-Disposition"), ";")[0]
		} else if isAttachment {
			altContent.Type = "Att"
			parts := strings.Split(headers.Get("Content-Disposition"), ";")
			altContent.Layout = parts[0]
			altContent.Name = strings.Split(parts[1], "=")[1]
		} else {
			altContent.Type = "Main"
			mail2.Body = append(mail2.Body, altContent.ID)
		}
		mailBox.Contents[altContent.ID] = altContent
		mailBox.Mails[mail2.ID] = mail2
	}
	return nil
}
