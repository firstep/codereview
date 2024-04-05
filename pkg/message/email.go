package message

import (
	"fmt"
	"io"
	"net/smtp"
	"strings"

	"github.com/firstep/aries/config"
	"github.com/jordan-wright/email"
)

type Attach struct {
	File     io.Reader
	FileName string
	MiniType string
}

type loginAuth struct {
	username, password string
}

const (
	TypeText = iota + 1
	TypeHTML
)

var (
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	smtpSender   string
)

func init() {
	smtpHost = config.GetString("smtp.host", "")
	if smtpHost == "" {
		panic("SMTP host is required")
	}

	smtpPort = config.GetInt("smtp.port", 0)
	if smtpPort == 0 {
		panic("SMTP port is required")
	}

	smtpUsername = config.GetString("smtp.username", "")
	if smtpUsername == "" {
		panic("SMTP username is required")
	}

	smtpPassword = config.GetString("smtp.password", "")
	if smtpPassword == "" {
		panic("SMTP password is required")
	}

	smtpSender = config.GetString("smtp.sender", "")
	if smtpSender == "" {
		panic("SMTP sender is required")
	}
}

func SendEmailWithText(subject string, receiver []string, content string, attachs ...Attach) error {
	return SendEmail(subject, receiver, content, TypeText, attachs...)
}

func SendEmailWithHTML(subject string, receiver []string, content string, attachs ...Attach) error {
	return SendEmail(subject, receiver, content, TypeHTML, attachs...)
}

func SendEmail(subject string, receiver []string, content string, contentType int, attachs ...Attach) error {
	e := email.NewEmail()

	e.From = smtpSender
	e.To = receiver
	e.Subject = subject
	if contentType == TypeText {
		e.Text = []byte(content)
	} else {
		e.HTML = []byte(content)
	}

	for _, attach := range attachs {
		e.Attach(attach.File, attach.FileName, attach.MiniType)
	}

	return e.Send(fmt.Sprintf("%s:%d", smtpHost, smtpPort), LoginAuth(smtpUsername, smtpPassword))
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	command := string(fromServer)
	command = strings.TrimSpace(command)
	command = strings.TrimSuffix(command, ":")
	command = strings.ToLower(command)

	if more {
		if command == "username" {
			return []byte(fmt.Sprintf("%s", a.username)), nil
		} else if command == "password" {
			return []byte(fmt.Sprintf("%s", a.password)), nil
		} else {
			return nil, fmt.Errorf("unexpected server challenge: %s", command)
		}
	}
	return nil, nil
}
