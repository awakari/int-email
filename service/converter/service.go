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
	Convert(src io.Reader, dst *pb.CloudEvent, internal bool) (err error)
}

type svc struct {
	evtType           string
	htmlPolicy        *bluemonday.Policy
	writerInternalCfg config.WriterInternalConfig
}

const ceKeyLenMax = 20
const ceKeyObjectUrl = "objecturl"
const ceKeyTime = "time"
const ceKeyAttContentIds = "attachmentcids"
const ceKeyAttContentTypes = "attachmentctypes"
const ceKeyAttFileNames = "attachmentfilenames"
const ceSpecVersion = "1.0"

var ErrParse = errors.New("failed to parse message")
var headerBlacklist = map[string]bool{
	"bcc":                  true,
	"cc":                   true,
	"deliveredto":          true,
	"deliverto":            true,
	"dkimsignature":        true,
	"from":                 true,
	"listunsubscribe":      true,
	"received":             true,
	"returnpath":           true,
	"to":                   true,
	"xgmmessagestate":      true,
	"xgoogledkimsignature": true,
	"xgooglesmtpsource":    true,
	"xmailgunbatchid":      true,
	"xmailgunsendingip":    true,
	"xmailgunsid":          true,
	"xmailgunvariables":    true,
	"xreceived":            true,
}
var reUrlTail = regexp.MustCompile(`\?[a-zA-Z0-9_\-]+=[a-zA-Z0-9_\-~.%&/#+]*`)

func NewConverter(evtType string, htmlPolicy *bluemonday.Policy, writerInternalCfg config.WriterInternalConfig) Service {
	return svc{
		evtType:           evtType,
		htmlPolicy:        htmlPolicy,
		writerInternalCfg: writerInternalCfg,
	}
}

func (c svc) Convert(src io.Reader, dst *pb.CloudEvent, internal bool) (err error) {
	var e *enmime.Envelope
	e, err = enmime.ReadEnvelope(src)
	switch err {
	case nil:
		err = c.convert(e, dst, internal)
	default:
		err = fmt.Errorf("%w: %s", ErrParse, err)
	}
	return
}

func (c svc) convert(src *enmime.Envelope, dst *pb.CloudEvent, internal bool) (err error) {

	for _, k := range src.GetHeaderKeys() {
		v := src.GetHeader(k)
		ceKey := c.convertHeaderKey(k)
		switch ceKey {
		case "date":
			var t time.Time
			t, err = time.Parse(time.RFC1123Z, v)
			switch err {
			case nil:
				dst.Attributes[ceKeyTime] = &pb.CloudEventAttributeValue{
					Attr: &pb.CloudEventAttributeValue_CeTimestamp{
						CeTimestamp: timestamppb.New(t),
					},
				}
			default:
				err = fmt.Errorf("%w: %s", ErrParse, err)
			}
		case "messageid":
			objectUrl := c.convertAddr(v)
			dst.Attributes[ceKeyObjectUrl] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeUri{
					CeUri: objectUrl,
				},
			}
		case "listurl":
			dst.Source = c.convertAddr(v)
		default:
			if internal || !headerBlacklist[ceKey] && v != "" {
				v = c.convertAddr(v)
				dst.Attributes[ceKey] = &pb.CloudEventAttributeValue{
					Attr: &pb.CloudEventAttributeValue_CeString{
						CeString: v,
					},
				}
			}
		}
		if err != nil {
			break
		}
	}

	if err == nil {
		if src.Text != "" {
			dst.Data = &pb.CloudEvent_TextData{
				TextData: src.Text,
			}
		}
		if src.HTML != "" {
			err = c.handleHtml(src.HTML, dst)
			if err == nil {
				switch internal {
				case true:
					dst.Data = &pb.CloudEvent_TextData{
						TextData: src.HTML,
					}
				default:
					txt := reUrlTail.ReplaceAllString(src.HTML, "\"")
					dst.Data = &pb.CloudEvent_TextData{
						TextData: c.htmlPolicy.Sanitize(txt),
					}
				}
			}
		}
		if err == nil && dst.Data == nil {
			err = fmt.Errorf("%w: %s", ErrParse, "no text data")
		}
	}

	if err == nil {
		if dst.Attributes[ceKeyTime] == nil {
			err = fmt.Errorf("%w: %s", ErrParse, "no message date in the source data")
		}
		if dst.Attributes[ceKeyObjectUrl] == nil {
			err = fmt.Errorf("%w: %s", ErrParse, "no message in the source data")
		}
	}

	if err == nil {

		dst.Id = ksuid.New().String()
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
	}

	if internal {
		dst.Attributes[c.writerInternalCfg.Name] = &pb.CloudEventAttributeValue{
			Attr: &pb.CloudEventAttributeValue_CeInteger{
				CeInteger: c.writerInternalCfg.Value,
			},
		}
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

func (c svc) handleHtml(src string, evt *pb.CloudEvent) (err error) {
	var doc *goquery.Document
	doc, err = goquery.NewDocumentFromReader(strings.NewReader(src))
	if err != nil {
		err = fmt.Errorf("%w: %s", ErrParse, err)
	}
	if err == nil {
		s := doc.Find("a.email-button-outline")
		for _, n := range s.Nodes {
			var urlOrig string
			for _, a := range n.Attr {
				if a.Key == "href" {
					urlOrig = a.Val
					break
				}
			}
			if urlOrig != "" {
				urlEnd := strings.Index(urlOrig, "?")
				if urlEnd > 0 {
					urlOrig = urlOrig[:urlEnd]
				}
				evt.Attributes[ceKeyObjectUrl] = &pb.CloudEventAttributeValue{
					Attr: &pb.CloudEventAttributeValue_CeUri{
						CeUri: urlOrig,
					},
				}
				break
			}
		}
	}
	return
}
