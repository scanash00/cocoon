package server

import (
	"fmt"
	"strings"
	"time"
)

func (s *Server) sendWelcomeMail(email, handle string) error {
	if s.mail == nil {
		return nil
	}

	s.mailLk.Lock()
	defer s.mailLk.Unlock()

	s.mail.To(email)
	s.mail.Subject("Welcome to " + s.config.Hostname)
	s.mail.Plain().Set(fmt.Sprintf("Welcome to %s! Your handle is %s.", email, handle))

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

	s.mail.To(email)
	s.mail.Subject("Password reset for " + s.config.Hostname)
	s.mail.Plain().Set(fmt.Sprintf("Hello %s. Your password reset code is %s. This code will expire in ten minutes.", handle, code))

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

	s.mail.To(email)
	s.mail.Subject("PLC token for " + s.config.Hostname)
	s.mail.Plain().Set(fmt.Sprintf("Hello %s. Your PLC operation code is %s. This code will expire in ten minutes.", handle, code))

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

	s.mail.To(email)
	s.mail.Subject("Email update for " + s.config.Hostname)
	s.mail.Plain().Set(fmt.Sprintf("Hello %s. Your email update code is %s. This code will expire in ten minutes.", handle, code))

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

	s.mail.To(email)
	s.mail.Subject("Email verification for " + s.config.Hostname)
	s.mail.Plain().Set(fmt.Sprintf("Hello %s. Your email verification code is %s. This code will expire in ten minutes.", handle, code))

	if err := s.mail.Send(); err != nil {
		return err
	}

	return nil
}
