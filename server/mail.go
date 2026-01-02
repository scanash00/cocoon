package server

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed templates/email/*
var emailTemplateFS embed.FS

type emailData struct {
	Subject  string
	Hostname string
	Handle   string
	Code     string
}

func (s *Server) renderEmailTemplate(templateName string, data emailData) (string, error) {
	if s.config.Version == "dev" {
		tmpl, err := template.ParseFiles(
			"server/templates/email/base.html",
			"server/templates/email/"+templateName+".html",
		)
		if err != nil {
			return "", err
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	tmpl, err := template.ParseFS(emailTemplateFS,
		"templates/email/base.html",
		"templates/email/"+templateName+".html",
	)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *Server) sendWelcomeMail(email, handle string) error {
	if s.mail == nil {
		return nil
	}

	s.mailLk.Lock()
	defer s.mailLk.Unlock()

	data := emailData{
		Subject:  "Welcome to " + s.config.Hostname,
		Hostname: s.config.Hostname,
		Handle:   handle,
	}

	htmlBody, err := s.renderEmailTemplate("welcome", data)
	if err != nil {
		s.logger.Error("failed to render welcome email template", "error", err)
		htmlBody = ""
	}

	s.mail.To(email)
	s.mail.Subject(data.Subject)
	s.mail.Plain().Set("Welcome to " + s.config.Hostname + "! Your handle is " + handle + ".")
	if htmlBody != "" {
		s.mail.HTML().Set(htmlBody)
	}

	if err := s.mail.Send(); err != nil {
		return err
	}

	return nil
}

func (s *Server) sendAppealNotice(appellantDid, appellantHandle string, takedownComment *string, reasonType, appealText, subjectJSON string) error {
	if s.mail == nil {
		return nil
	}

	s.mailLk.Lock()
	defer s.mailLk.Unlock()

	parts := []string{}
	parts = append(parts, "A takedown account submitted an appeal/report.")
	parts = append(parts, fmt.Sprintf("\nAppellant DID: %s", appellantDid))
	if appellantHandle != "" {
		parts = append(parts, fmt.Sprintf("Appellant handle: %s", appellantHandle))
	}
	if takedownComment != nil && *takedownComment != "" {
		parts = append(parts, fmt.Sprintf("\nOriginal takedown comment:\n%s", *takedownComment))
	}
	if reasonType != "" {
		parts = append(parts, fmt.Sprintf("\nReport reasonType: %s", reasonType))
	}
	if appealText != "" {
		parts = append(parts, fmt.Sprintf("\nAppeal text:\n%s", appealText))
	}
	if subjectJSON != "" {
		parts = append(parts, fmt.Sprintf("\nReport subject:\n%s", subjectJSON))
	}
	parts = append(parts, fmt.Sprintf("\nHost: %s", s.config.Hostname))

	msg := strings.Join(parts, "\n")

	s.mail.To(s.config.ContactEmail)
	s.mail.Subject("Account appeal received for " + s.config.Hostname)
	s.mail.Plain().Set(msg)

	if err := s.mail.Send(); err != nil {
		return err
	}

	return nil
}

func (s *Server) sendTakedownNotice(email, handle string, reason *string, expiresAt *time.Time) error {
	if s.mail == nil {
		return nil
	}

	s.mailLk.Lock()
	defer s.mailLk.Unlock()

	msg := fmt.Sprintf("Hello %s,\n\n", handle)
	msg += fmt.Sprintf("Your account on %s has been suspended.", s.config.Hostname)
	if reason != nil && *reason != "" {
		msg += fmt.Sprintf("\n\nViolation/Reason:\n%s", *reason)
	}
	if expiresAt != nil {
		msg += fmt.Sprintf("\n\nSuspension period:\nUntil %s", expiresAt.UTC().Format(time.RFC3339))
	} else {
		msg += "\n\nSuspension period:\nPermanent (no end time set)"
	}
	msg += fmt.Sprintf("\n\nIf you believe this decision was made in error, you can appeal by contacting: %s\n", s.config.ContactEmail)

	s.mail.To(email)
	s.mail.Subject("Account suspension notice for " + s.config.Hostname)
	s.mail.Plain().Set(msg)

	if err := s.mail.Send(); err != nil {
		return err
	}

	return nil
}

func (s *Server) sendPasswordReset(email, handle, code string) error {
	if s.mail == nil {
		return nil
	}

	s.mailLk.Lock()
	defer s.mailLk.Unlock()

	data := emailData{
		Subject:  "Password reset for " + s.config.Hostname,
		Hostname: s.config.Hostname,
		Handle:   handle,
		Code:     code,
	}

	htmlBody, err := s.renderEmailTemplate("password_reset", data)
	if err != nil {
		s.logger.Error("failed to render password reset email template", "error", err)
		htmlBody = ""
	}

	s.mail.To(email)
	s.mail.Subject(data.Subject)
	s.mail.Plain().Set("Hello " + handle + ". Your password reset code is " + code + ". This code will expire in ten minutes.")
	if htmlBody != "" {
		s.mail.HTML().Set(htmlBody)
	}

	if err := s.mail.Send(); err != nil {
		return err
	}

	return nil
}

func (s *Server) sendPlcTokenReset(email, handle, code string) error {
	if s.mail == nil {
		return nil
	}

	s.mailLk.Lock()
	defer s.mailLk.Unlock()

	data := emailData{
		Subject:  "PLC token for " + s.config.Hostname,
		Hostname: s.config.Hostname,
		Handle:   handle,
		Code:     code,
	}

	htmlBody, err := s.renderEmailTemplate("plc_token", data)
	if err != nil {
		s.logger.Error("failed to render PLC token email template", "error", err)
		htmlBody = ""
	}

	s.mail.To(email)
	s.mail.Subject(data.Subject)
	s.mail.Plain().Set("Hello " + handle + ". Your PLC operation code is " + code + ". This code will expire in ten minutes.")
	if htmlBody != "" {
		s.mail.HTML().Set(htmlBody)
	}

	if err := s.mail.Send(); err != nil {
		return err
	}

	return nil
}

func (s *Server) sendEmailUpdate(email, handle, code string) error {
	if s.mail == nil {
		return nil
	}

	s.mailLk.Lock()
	defer s.mailLk.Unlock()

	data := emailData{
		Subject:  "Email update for " + s.config.Hostname,
		Hostname: s.config.Hostname,
		Handle:   handle,
		Code:     code,
	}

	htmlBody, err := s.renderEmailTemplate("email_update", data)
	if err != nil {
		s.logger.Error("failed to render email update template", "error", err)
		htmlBody = ""
	}

	s.mail.To(email)
	s.mail.Subject(data.Subject)
	s.mail.Plain().Set("Hello " + handle + ". Your email update code is " + code + ". This code will expire in ten minutes.")
	if htmlBody != "" {
		s.mail.HTML().Set(htmlBody)
	}

	if err := s.mail.Send(); err != nil {
		return err
	}

	return nil
}

func (s *Server) sendEmailVerification(email, handle, code string) error {
	if s.mail == nil {
		return nil
	}

	s.mailLk.Lock()
	defer s.mailLk.Unlock()

	data := emailData{
		Subject:  "Email verification for " + s.config.Hostname,
		Hostname: s.config.Hostname,
		Handle:   handle,
		Code:     code,
	}

	htmlBody, err := s.renderEmailTemplate("email_verification", data)
	if err != nil {
		s.logger.Error("failed to render email verification template", "error", err)
		htmlBody = ""
	}

	s.mail.To(email)
	s.mail.Subject(data.Subject)
	s.mail.Plain().Set("Hello " + handle + ". Your email verification code is " + code + ". This code will expire in ten minutes.")
	if htmlBody != "" {
		s.mail.HTML().Set(htmlBody)
	}

	if err := s.mail.Send(); err != nil {
		return err
	}

	return nil
}
