package email

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"
)

var wordDecoder = new(mime.WordDecoder)

func decodeHeader(s string) string {
	decoded, err := wordDecoder.DecodeHeader(s)
	if err != nil {
		return s
	}
	return decoded
}

type ParsedEmail struct {
	From        string
	To          []string
	Subject     string
	BodyText    string
	BodyHTML    string
	Attachments []ParsedAttachment
	Headers     map[string][]string
}

type ParsedAttachment struct {
	Filename    string
	Content     []byte
	ContentType string
	ContentID   string
	Size        int
}

func ParseMIME(rawEmail []byte) (*ParsedEmail, error) {
	if len(rawEmail) == 0 {
		return nil, fmt.Errorf("empty email data")
	}

	msg, err := mail.ReadMessage(bytes.NewReader(rawEmail))
	if err != nil {
		return nil, fmt.Errorf("parse email headers: %w", err)
	}

	result := &ParsedEmail{
		Headers: map[string][]string{},
	}

	result.From = parseAddressList(msg.Header.Get("From"))
	result.To = parseAddressLists(msg.Header.Get("To"), msg.Header.Get("Cc"))
	result.Subject = decodeHeader(msg.Header.Get("Subject"))

	for k, v := range msg.Header {
		result.Headers[k] = v
	}

	contentType := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = "text/plain"
		params = map[string]string{}
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			body, _ := io.ReadAll(msg.Body)
			result.BodyText = string(body)
			return result, nil
		}
		if err := parseMultipart(msg.Body, boundary, result); err != nil {
			return nil, fmt.Errorf("parse multipart: %w", err)
		}
	} else if strings.HasPrefix(mediaType, "text/plain") {
		body, _ := io.ReadAll(msg.Body)
		result.BodyText = string(body)
	} else if strings.HasPrefix(mediaType, "text/html") {
		body, _ := io.ReadAll(msg.Body)
		result.BodyHTML = string(body)
	} else {
		body, _ := io.ReadAll(msg.Body)
		result.Attachments = append(result.Attachments, ParsedAttachment{
			ContentType: mediaType,
			Content:     body,
			Size:        len(body),
		})
	}

	return result, nil
}

func parseMultipart(body io.Reader, boundary string, result *ParsedEmail) error {
	reader := multipart.NewReader(body, boundary)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read part: %w", err)
		}

		contentType := part.Header.Get("Content-Type")
		mediaType, params, _ := mime.ParseMediaType(contentType)

		data, err := io.ReadAll(part)
		if err != nil {
			_ = part.Close()
			continue
		}
		_ = part.Close()

		disposition, dispParams, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		isAttachment := disposition == "attachment"
		filename := extractFilename(part.Header, dispParams, params)

		if strings.HasPrefix(mediaType, "multipart/") {
			subBoundary := params["boundary"]
			if subBoundary != "" {
				if err := parseMultipart(bytes.NewReader(data), subBoundary, result); err != nil {
					continue
				}
			}
		} else if isAttachment || (filename != "" && !strings.HasPrefix(mediaType, "text/")) {
			result.Attachments = append(result.Attachments, ParsedAttachment{
				Filename:    filename,
				Content:     data,
				ContentType: mediaType,
				ContentID:   extractContentID(part.Header),
				Size:        len(data),
			})
		} else if strings.HasPrefix(mediaType, "text/plain") && result.BodyText == "" {
			result.BodyText = string(data)
		} else if strings.HasPrefix(mediaType, "text/html") && result.BodyHTML == "" {
			result.BodyHTML = string(data)
		} else if filename != "" {
			result.Attachments = append(result.Attachments, ParsedAttachment{
				Filename:    filename,
				Content:     data,
				ContentType: mediaType,
				ContentID:   extractContentID(part.Header),
				Size:        len(data),
			})
		}
	}
	return nil
}

func extractFilename(h textproto.MIMEHeader, dispParams, contentTypeParams map[string]string) string {
	if name := dispParams["filename"]; name != "" {
		return decodeHeader(name)
	}
	if name := contentTypeParams["name"]; name != "" {
		return decodeHeader(name)
	}
	return ""
}

func extractContentID(h textproto.MIMEHeader) string {
	cid := h.Get("Content-ID")
	cid = strings.Trim(cid, "<>")
	return cid
}

func parseAddressList(addr string) string {
	if addr == "" {
		return ""
	}
	addrs, err := mail.ParseAddress(addr)
	if err != nil {
		return addr
	}
	if addrs.Name != "" {
		return addrs.Name + " <" + addrs.Address + ">"
	}
	return addrs.Address
}

func parseAddressLists(fields ...string) []string {
	var result []string
	for _, field := range fields {
		if field == "" {
			continue
		}
		addrs, err := mail.ParseAddressList(field)
		if err != nil {
			result = append(result, field)
			continue
		}
		for _, a := range addrs {
			result = append(result, a.Address)
		}
	}
	return result
}
