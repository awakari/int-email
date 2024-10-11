package converter

import (
	"errors"
	"fmt"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/jhillyerd/enmime"
	"github.com/segmentio/ksuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"strings"
	"time"
)

type Service interface {
	Convert(src io.Reader, dst *pb.CloudEvent) (err error)
}

type svc struct {
	evtType string
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
	"received":             true,
	"returnpath":           true,
	"to":                   true,
	"xgmmessagestate":      true,
	"xgoogledkimsignature": true,
	"xgooglesmtpsource":    true,
	"xreceived":            true,
}

func NewConverter(evtType string) Service {
	return svc{
		evtType: evtType,
	}
}

func (c svc) Convert(src io.Reader, dst *pb.CloudEvent) (err error) {
	var e *enmime.Envelope
	e, err = enmime.ReadEnvelope(src)
	switch err {
	case nil:
		err = c.convert(e, dst)
	default:
		err = fmt.Errorf("%w: %s", ErrParse, err)
	}
	return
}

func (c svc) convert(src *enmime.Envelope, dst *pb.CloudEvent) (err error) {

	if src.Text != "" {
		dst.Data = &pb.CloudEvent_TextData{
			TextData: src.Text,
		}
	}
	if src.HTML != "" {
		dst.Data = &pb.CloudEvent_TextData{
			TextData: src.HTML,
		}
	}
	if dst.Data == nil {
		err = fmt.Errorf("%w: %s", ErrParse, "no text data")
	}

	if err == nil {
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
				objectUrl := v
				if strings.HasPrefix(objectUrl, "<") {
					objectUrl = objectUrl[1:]
				}
				if strings.HasSuffix(objectUrl, ">") {
					objectUrl = objectUrl[:len(objectUrl)-1]
				}
				dst.Attributes[ceKeyObjectUrl] = &pb.CloudEventAttributeValue{
					Attr: &pb.CloudEventAttributeValue_CeUri{
						CeUri: objectUrl,
					},
				}
			default:
				if !headerBlacklist[ceKey] {
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

	return
}

func (c svc) convertHeaderKey(src string) (dst string) {
	dst = strings.Replace(strings.ToLower(src), "-", "", -1)
	if len(dst) > ceKeyLenMax {
		dst = dst[:ceKeyLenMax]
	}
	return
}
