package converter

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/awakari/int-email/config"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/jhillyerd/enmime"
	"github.com/microcosm-cc/bluemonday"
	"github.com/segmentio/ksuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"regexp"
	"strings"
	"time"
)

type Service interface {
	Convert(src io.Reader, dst *pb.CloudEvent, from string, internal bool) (err error)
}

type svc struct {
	evtType           string
	htmlPolicy        *bluemonday.Policy
	writerInternalCfg config.WriterInternalConfig
	rcptsPublish      map[string]bool
	truncUrlQuery     bool
}

const ceKeyLenMax = 20
const ceKeyObjectUrl = "objecturl"
const ceKeySummary = "summary"
const ceKeyTime = "time"

const ceKeyAttContentIds = "attachmentcids"
const ceKeyAttContentTypes = "attachmentctypes"
const ceKeyAttFileNames = "attachmentfilenames"

const ceSpecVersion = "1.0"

var ErrParse = errors.New("failed to parse message")
var headerWhiteList = map[string]bool{
	"contenttransferencod": true,
	"contenttype":          true,
	"feedbackid":           true,
	"inreplyto":            true,
	"listhelp":             true,
	"listhelpurl":          true,
	"listid":               true,
	"listowner":            true,
	"listpost":             true,
	"listsubscribe":        true,
	"listurl":              true,
	"mimeversion":          true,
	"precedence":           true,
	"references":           true,
	"sender":               true,
	"subject":              true,
	"xcompanyid":           true,
	"xemailcategory":       true,
	"xfeedbackid":          true,
	"xmailer":              true,
	"xmailguntag":          true,
	"xorganization":        true,
	"xpriority":            true,
	"xreportabuse":         true,
	"xvirusscanned":        true,
}
var reUrlQuery = regexp.MustCompile(`\?[a-zA-Z0-9_\-]+=[a-zA-Z0-9_\-~.%&/#+]*`)

func NewConverter(evtType string, htmlPolicy *bluemonday.Policy, writerInternalCfg config.WriterInternalConfig, rcptsPublish map[string]bool, truncUrlQuery bool) Service {
	return svc{
		evtType:           evtType,
		htmlPolicy:        htmlPolicy,
		writerInternalCfg: writerInternalCfg,
		rcptsPublish:      rcptsPublish,
		truncUrlQuery:     truncUrlQuery,
	}
}

func (c svc) Convert(src io.Reader, dst *pb.CloudEvent, from string, internal bool) (err error) {
	var e *enmime.Envelope
	e, err = enmime.ReadEnvelope(src)
	switch err {
	case nil:
		err = c.convert(e, dst, from, internal)
	default:
		err = fmt.Errorf("%w: %s", ErrParse, err)
	}
	return
}

func (c svc) convert(src *enmime.Envelope, dst *pb.CloudEvent, from string, internal bool) (err error) {
	err = c.convertHeaders(src, dst, from, internal)
	if err == nil {
		err = c.convertBody(src, dst, internal)
	}
	if err == nil {
		c.convertAttachments(src, dst, from)
		if internal {
			dst.Attributes[c.writerInternalCfg.Name] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: c.writerInternalCfg.Value,
				},
			}
		}
	}
	return
}

func (c svc) convertHeaders(src *enmime.Envelope, dst *pb.CloudEvent, from string, internal bool) (err error) {
	for _, k := range src.GetHeaderKeys() {
		v := src.GetHeader(k)
		ceKey := c.convertHeaderKey(k)
		switch ceKey {
		case "date":
			t, tErr := time.Parse(time.RFC1123Z, v)
			if tErr == nil {
				dst.Attributes[ceKeyTime] = &pb.CloudEventAttributeValue{
					Attr: &pb.CloudEventAttributeValue_CeTimestamp{
						CeTimestamp: timestamppb.New(t),
					},
				}
			}
		case "from":
			fromActual := v
			addrPos := strings.Index(v, "<")
			if addrPos > 0 {
				fromActual = fromActual[addrPos:]
			}
			dst.Source = c.cleanRecipients(c.convertAddr(fromActual))
		case "listurl":
			dst.Source = c.cleanRecipients(c.convertAddr(v))
		case "messageid":
			dst.Attributes[ceKeyObjectUrl] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeUri{
					CeUri: c.cleanRecipients(c.convertAddr(v)),
				},
			}
		case "subject":
			dst.Attributes[ceKeySummary] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeString{
					CeString: c.cleanRecipients(v),
				},
			}
		default:
			switch {
			case internal, headerWhiteList[ceKey]:
				v = c.convertAddr(v)
				if v != "" {
					dst.Attributes[ceKey] = &pb.CloudEventAttributeValue{
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: c.cleanRecipients(v),
						},
					}
				}
			default:
				fmt.Printf("forbidden header from %s: %s=%s\n", from, k, v)
			}
		}
	}
	if dst.Attributes[ceKeyTime] == nil {
		dst.Attributes[ceKeyTime] = &pb.CloudEventAttributeValue{
			Attr: &pb.CloudEventAttributeValue_CeTimestamp{
				CeTimestamp: timestamppb.New(time.Now().UTC()),
			},
		}
	}
	if dst.Attributes[ceKeyObjectUrl] == nil {
		err = fmt.Errorf("%w: %s", ErrParse, "no message id in the source data")
	}
	return
}

func (c svc) convertHeaderKey(src string) (dst string) {
	dst = strings.Replace(strings.ToLower(src), "-", "", -1)
	if len(dst) > ceKeyLenMax {
		dst = dst[:ceKeyLenMax]
	}
	return
}

func (c svc) convertAddr(src string) (dst string) {
	dst = src
	if strings.HasPrefix(dst, "<") {
		dst = dst[1:]
		if strings.HasSuffix(dst, ">") {
			dst = dst[:len(dst)-1]
		}
	}
	urlEnd := strings.Index(dst, "?")
	if urlEnd > 0 {
		dst = dst[:urlEnd]
	}
	return
}

func (c svc) convertBody(src *enmime.Envelope, dst *pb.CloudEvent, internal bool) (err error) {
	var txt string
	if src.Text != "" {
		txt = src.Text
	}
	if src.HTML != "" {
		err = c.handleHtml(src.HTML, dst)
		if err == nil {
			txt = src.HTML
			if !internal {
				if c.truncUrlQuery {
					txt = reUrlQuery.ReplaceAllString(txt, "\"")
				}
				txt = c.htmlPolicy.Sanitize(txt)
			}
		}
	}
	if err == nil {
		switch txt {
		case "":
			err = fmt.Errorf("%w: %s", ErrParse, "no text data")
		default:
			dst.Data = &pb.CloudEvent_TextData{
				TextData: strings.TrimSpace(c.cleanRecipients(txt)),
			}
		}
	}
	return
}

func (c svc) handleHtml(src string, evt *pb.CloudEvent) (err error) {
	var doc *goquery.Document
	doc, err = goquery.NewDocumentFromReader(strings.NewReader(src))
	if err != nil {
		err = fmt.Errorf("%w: %s", ErrParse, err)
	}
	if err == nil {
		// quora
		s := doc.
			Find("td.answer_details").
			Find("a")
		c.handleUrlOriginal(s, evt, false)
		// substack
		s = doc.Find("a.email-button-outline")
		c.handleUrlOriginal(s, evt, true)
	}
	return
}

func (c svc) handleUrlOriginal(s *goquery.Selection, evt *pb.CloudEvent, trunc bool) {
	for _, n := range s.Nodes {
		var urlOrig string
		for _, a := range n.Attr {
			if a.Key == "href" {
				urlOrig = a.Val
				break
			}
		}
		if urlOrig != "" {
			if trunc {
				urlEnd := strings.Index(urlOrig, "?")
				if urlEnd > 0 {
					urlOrig = urlOrig[:urlEnd]
				}
			}
			evt.Attributes[ceKeyObjectUrl] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeUri{
					CeUri: urlOrig,
				},
			}
			break
		}
	}
	return
}

func (c svc) convertAttachments(src *enmime.Envelope, dst *pb.CloudEvent, from string) {
	dst.Id = ksuid.New().String()
	if dst.Source == "" {
		dst.Source = c.cleanRecipients(from)
	}
	dst.SpecVersion = ceSpecVersion
	dst.Type = c.evtType

	var parts []*enmime.Part
	parts = append(parts, src.Attachments...)
	parts = append(parts, src.Inlines...)
	parts = append(parts, src.OtherParts...)
	var contentIds []string
	var contentTypes []string
	var fileNames []string
	for _, p := range parts {
		contentIds = append(contentIds, p.ContentID)
		contentTypes = append(contentTypes, p.ContentType)
		fileNames = append(fileNames, p.FileName)
	}
	if len(parts) > 0 {
		dst.Attributes[ceKeyAttContentIds] = &pb.CloudEventAttributeValue{
			Attr: &pb.CloudEventAttributeValue_CeString{
				CeString: strings.Join(contentIds, ", "),
			},
		}
		dst.Attributes[ceKeyAttContentTypes] = &pb.CloudEventAttributeValue{
			Attr: &pb.CloudEventAttributeValue_CeString{
				CeString: strings.Join(contentTypes, ", "),
			},
		}
		dst.Attributes[ceKeyAttFileNames] = &pb.CloudEventAttributeValue{
			Attr: &pb.CloudEventAttributeValue_CeString{
				CeString: strings.Join(fileNames, ", "),
			},
		}
	}
	return
}

func (c svc) cleanRecipients(src string) (dst string) {
	dst = src
	for rcpt := range c.rcptsPublish {
		dst = strings.ReplaceAll(dst, rcpt+"@", "")
		dst = strings.ReplaceAll(dst, strings.ToLower(rcpt)+"@", "")
		dst = strings.ReplaceAll(dst, rcpt, "")
		dst = strings.ReplaceAll(dst, strings.ToLower(rcpt), "")
	}
	return
}
