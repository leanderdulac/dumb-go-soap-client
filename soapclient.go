/*Package soapclient is a small SOAP client largely based on gowsdl's generated functions.*/
package soapclient

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
)

/*SOAPEnvelope represents a SOAP envelope.

Aside from it, it Also allows for setting a XSI (XMLSchema-instance) namespace if the XSIXmlns field is set to it.*/
type SOAPEnvelope struct {
	XMLName  xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	XSIXmlns string   `xml:"xmlns:xsi,attr"`

	Header *SOAPHeader
	Body   SOAPBody
}

/*SOAPHeader represents a SOAP header.*/
type SOAPHeader struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"`
	Header  interface{}
}

/*SOAPBody represents a SOAP body.

When unmarshaled into, it carries either a "Fault" (if the SOAP response is faulted) or a "Content" with the SOAP response's body.*/
type SOAPBody struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`

	Fault   *SOAPFault  `xml:",omitempty"`
	Content interface{} `xml:",any"`
}

/*SOAPFault represents a SOAP fault.*/
type SOAPFault struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`

	Code   string `xml:"faultcode,omitempty"`
	String string `xml:"faultstring,omitempty"`
	Actor  string `xml:"faultactor,omitempty"`
	Detail string `xml:"detail,omitempty"`
}

/*A SOAPClient can perform SOAP requests to an endpoint.*/
type SOAPClient struct {
	endpoint string
}

/*Initialize a SOAPClient with a SOAP endpoint.*/
func New(endpoint string) *SOAPClient {
	return &SOAPClient{endpoint}
}

/*
Do a SOAP request given a client with the SOAP action and request supplied, returning a response.

The function will:
1. use encoding/xml to marshal the "request" interface into the SOAP request's body;
2. will send the SOAP request
3. will unmarshal the SOAP response's SOAP body to "response", which should an empty pointer to a marshalable struct.

This function returns an error on case of XML encoding errors, HTTP errors, an empty SOAP response
or a SOAP fault.

This function does not yet provide support for handling SOAP headers or SOAP faults.
*/
func (s *SOAPClient) Do(soapAction string, request, response interface{}) error {
	/* TODO: This function is very limited for a SOAP client. We should be
	able to send/receive headers, expose faults and have TLS options. */

	envelope := SOAPEnvelope{
		XSIXmlns: "http://www.w3.org/2001/XMLSchema-instance",
		Header:   nil,
	}
	envelope.Body.Content = request

	buffer := new(bytes.Buffer)
	encoder := xml.NewEncoder(buffer)

	if err := encoder.Encode(envelope); err != nil {
		return err
	}

	if err := encoder.Flush(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", s.endpoint, buffer)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "text/xml; charset=\"utf-8\"")
	req.Header.Add("SOAPAction", soapAction)
	req.Close = true

	client := &http.Client{}
	res, err := client.Do(req)

	if err != nil {
		return err
	}
	defer res.Body.Close()

	rawbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if len(rawbody) == 0 {
		return fmt.Errorf("received empty raw body")
	}

	respEnvelope := new(SOAPEnvelope)
	respEnvelope.Body = SOAPBody{Content: response}

	err = xml.Unmarshal(rawbody, respEnvelope)

	if err != nil {
		return err
	}

	fault := respEnvelope.Body.Fault

	if fault != nil {
		return fmt.Errorf("received SOAP fault with code " + fault.Code)
	}

	return nil
}
