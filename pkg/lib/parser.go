package ghosttohugo

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"strings"
)

// replaceLinks naively assumes that there are links to replace, and it assumes
// that the link replacement map is initialized.
func (c *Config) replaceInLink(href string) string {
	for k, v := range c.LinkReplacements {
		href = strings.ReplaceAll(href, k, v)
	}

	return href
}

// Used in tandem with an xml encoder to ensure the content of an XML node
// is accessible - for example:
//
//	`<a href="">Content</a>``
//
// would be usable as
//
//	el.Content
type el struct {
	XMLName xml.Name
	Content string `xml:",chardata"` // This will capture the text content
}

// ProcessHTML removes all height and width tags from an input xml string, as well
// as anything else needed in order to process the document.
func (c *Config) ProcessHTML(s string) (string, error) {
	b := bytes.NewBuffer([]byte(s))
	x := xml.NewDecoder(b)
	x.Strict = false
	x.AutoClose = xml.HTMLAutoClose
	x.Entity = xml.HTMLEntity

	var o bytes.Buffer
	xe := xml.NewEncoder(&o)

	for {
		t, err := x.Token()
		if err != nil { // EOF
			if err.Error() != "EOF" {
				return "", fmt.Errorf("token parsing error: %v", err.Error())
			}
			break
		}

		st, ok := t.(xml.StartElement)
		if ok {
			switch st.Name.Local {
			case "img":
				// var i img
				var i any
				err := x.DecodeElement(&i, &st)
				if err != nil {
					log.Printf("failed to decode img element: %v", err.Error())
					continue
				}

				if len(st.Attr) != 0 {
					for j := len(st.Attr) - 1; j >= 0; j-- {
						switch st.Attr[j].Name.Local {
						case "height":
							// log.Printf("setting img height to empty")
							st.Attr[j].Value = ""
						case "width":
							// log.Printf("setting img width to empty")
							st.Attr[j].Value = ""
						}
					}
				}
				// write the modified token to the to-be-returned modified xml document
				err = xe.EncodeToken(st)
				if err != nil {
					return "", fmt.Errorf("failed to re-encode token: %w", err)
				}
				// Manually close img tags
				err = xe.EncodeToken(xml.EndElement{Name: st.Name})
				if err != nil {
					return "", fmt.Errorf("failed to encode end token: %w", err)
				}

				continue
			case "a":
				if c.ReplaceLinks && len(st.Attr) != 0 {
					var i el
					err := x.DecodeElement(&i, &st)
					if err != nil {
						log.Printf("failed to decode img element: %v", err.Error())
						continue
					}

					for j := len(st.Attr) - 1; j >= 0; j-- {
						switch st.Attr[j].Name.Local {
						case "href":
							st.Attr[j].Value = c.replaceInLink(st.Attr[j].Value)
						}
					}

					// write the modified token to the to-be-returned modified xml document
					err = xe.EncodeToken(st)
					if err != nil {
						return "", fmt.Errorf("failed to re-encode token: %w", err)
					}

					err = xe.EncodeToken(xml.CharData([]byte(i.Content)))
					if err != nil {
						return "", fmt.Errorf("failed to re-encode a content token: %w", err)
					}

					// Manually close a tags
					err = xe.EncodeToken(xml.EndElement{Name: st.Name})
					if err != nil {
						return "", fmt.Errorf("failed to encode end token: %w", err)
					}

					continue
				}
			}
			// write the modified token to the to-be-returned modified xml document
			err = xe.EncodeToken(st)
			if err != nil {
				return "", fmt.Errorf("failed to re-encode token: %w", err)
			}
		} else {
			// write the modified token to the to-be-returned modified xml document
			err = xe.EncodeToken(t)
			if err != nil {
				return "", fmt.Errorf("failed to re-encode token: %w", err)
			}
		}

	}

	err := xe.Flush()
	if err != nil {
		return "", fmt.Errorf("failed to flush token re-encoder: %w", err)
	}

	return o.String(), nil
}
