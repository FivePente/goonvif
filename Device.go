package goonvif

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/use-go/goonvif/device"
	"github.com/use-go/goonvif/gosoap"
	"github.com/use-go/goonvif/networking"
)

//Xlmns XML Scheam
var Xlmns = map[string]string{
	"xsi":          "http://www.w3.org/2001/XMLSchema-instance",
	"xsd":          "http://www.w3.org/2001/XMLSchema",
	"c14n":         "http://www.w3.org/2001/10/xml-exc-c14n#",
	"wsu":          "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd",
	"wsc":          "http://schemas.xmlsoap.org/ws/2005/02/sc",
	"xenc":         "http://www.w3.org/2001/04/xmlenc#",
	"ds":           "http://www.w3.org/2000/09/xmldsig#",
	"wsse":         "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd",
	"chan":         "http://schemas.microsoft.com/ws/2005/02/duplex",
	"wsa5":         "http://www.w3.org/2005/08/addressing",
	"h":            "http://tempuri.org/h.xsd",
	"xmime":        "http://tempuri.org/xmime.xsd",
	"xop":          "http://www.w3.org/2004/08/xop/include",
	"tt":           "http://www.onvif.org/ver10/schema",
	"wsrfbf":       "http://docs.oasis-open.org/wsrf/bf-2",
	"wstop":        "http://docs.oasis-open.org/wsn/t-1",
	"wsrfr":        "http://docs.oasis-open.org/wsrf/r-2",
	"tds":          "http://www.onvif.org/ver10/device/wsdl",
	"tev":          "http://www.onvif.org/ver10/events/wsdl",
	"wsnt":         "http://docs.oasis-open.org/wsn/b-2",
	"tmd":          "http://www.onvif.org/ver10/deviceIO/wsdl",
	"tptz":         "http://www.onvif.org/ver20/ptz/wsdl",
	"trt":          "http://www.onvif.org/ver10/media/wsdl",
	"tns1":         "http://www.onvif.org/ver10/topics",
	"timg":         "http://www.onvif.org/ver20/imaging/wsdl",
	"tan":          "http://www.onvif.org/ver20/analytics/wsdl",
	"wsa":          "http://www.w3.org/2004/08/addressing",
	"wsntw":        "http://docs.oasis-open.org/wsn/bw-2",
	"wsrf-rw":      "http://docs.oasis-open.org/wsrf/rw-2",
	"wsaw":         "http://www.w3.org/2006/05/addressing/wsdl",
	"onvif":        "http://www.onvif.org/ver10/schema",
	"tnshoneywell": "http://www.honeywell.com/acs/security",
	"trc":          "http://www.onvif.org/ver10/recording/wsdl",
	"tse":          "http://www.onvif.org/ver10/search/wsdl",
	"trp":          "http://www.onvif.org/ver10/replay/wsdl",
}

//DeviceType alias for int
type DeviceType int

// Onvif Device Type
const (
	NVD DeviceType = iota
	NVS
	NVA
	NVT
)

func (devType DeviceType) String() string {
	stringRepresentation := []string{
		"NetworkVideoDisplay",
		"NetworkVideoStorage",
		"NetworkVideoAnalytics",
		"NetworkVideoTransmitter",
	}
	i := uint8(devType)
	switch {
	case i <= uint8(NVT):
		return stringRepresentation[i]
	default:
		return strconv.Itoa(int(i))
	}
}

//GetServices return available endpoints
func (dev *Device) GetServices() map[string]string {
	return dev.endpoints
}

func readResponse(resp *http.Response) string {
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (dev *Device) getSupportedServices(resp *http.Response) {

	doc := etree.NewDocument()
	data, _ := ioutil.ReadAll(resp.Body)
	if err := doc.ReadFromBytes(data); err != nil {
		return
	}
	services := doc.FindElements("./Envelope/Body/GetCapabilitiesResponse/Capabilities/*/XAddr")
	for _, j := range services {
		dev.addEndpoint(j.Parent().Tag, j.Text())
	}
}

//NewDevice function construct a ONVIF Device entity
func NewDevice(xaddr string) (*Device, error) {
	dev := new(Device)
	dev.xaddr = xaddr
	dev.endpoints = make(map[string]string)
	dev.addEndpoint("Device", "http://"+xaddr+"/onvif/device_service")
	dev.addEndpoint("Search", "http://"+xaddr+"/onvif/Search_service")
	dev.addEndpoint("Recording", "http://"+xaddr+"/onvif/recording_service")
	dev.addEndpoint("Replay", "http://"+xaddr+"/onvif/replay_service")

	getCapabilities := device.GetCapabilities{Category: "All"}

	resp, err := dev.CallMethod(getCapabilities, nil)
	// fmt.Println(resp.Request.Host)
	// fmt.Println(readResponse(resp))
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errors.New("camera is not available at " + xaddr + " or it does not support ONVIF services")
	}

	dev.getSupportedServices(resp)
	return dev, nil
}

//NewDeviceWithAuth function construct a ONVIF Device entity with username and password
func NewDeviceWithAuth(xaddr, username, password string) (*Device, error) {
	dev := new(Device)
	dev.xaddr = xaddr
	dev.endpoints = make(map[string]string)
	dev.addEndpoint("Device", "http://"+xaddr+"/onvif/device_service")
	dev.addEndpoint("Search", "http://"+xaddr+"/onvif/Search_service")
	dev.addEndpoint("Recording", "http://"+xaddr+"/onvif/recording_service")
	dev.addEndpoint("Replay", "http://"+xaddr+"/onvif/replay_service")

	dev.Authenticate(username, password)

	getCapabilities := device.GetCapabilities{Category: "All"}

	resp, err := dev.CallMethod(getCapabilities, nil)
	// fmt.Println(resp.Request.Host)
	// fmt.Println(readResponse(resp))
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintln("GetCapabilitier failed at:", xaddr, err))
	}

	dev.getSupportedServices(resp)
	return dev, nil
}

func (dev *Device) addEndpoint(Key, Value string) {

	//use lowCaseKey
	//make key having ability to handle Mixed Case for Different vendor devcie (e.g. Events EVENTS, events)
	lowCaseKey := strings.ToLower(Key)
	dev.endpoints[lowCaseKey] = Value
}

//Authenticate function authenticate client in the ONVIF Device.
//Function takes <username> and <password> params.
//You should use this function to allow authorized requests to the ONVIF Device
//To change auth data call this function again.
func (dev *Device) Authenticate(username, password string) {
	dev.login = username
	dev.password = password
}

//GetEndpoint returns specific ONVIF service endpoint address
func (dev *Device) GetEndpoint(name string) string {
	return dev.endpoints[name]
}

func buildMethodSOAP(msg string) (gosoap.SoapMessage, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(msg); err != nil {
		//log.Println("Got error")

		return "", err
	}
	element := doc.Root()

	soap := gosoap.NewEmptySOAP()
	soap.AddBodyContent(element)
	//soap.AddRootNamespace("onvif", "http://www.onvif.org/ver10/device/wsdl")

	return soap, nil
}

//getEndpoint functions get the target service endpoint in a better way
func (dev *Device) getEndpoint(endpoint string) (string, error) {
	// common condition, endpointMark in map we use this.
	if endpointURL, bFound := dev.endpoints[endpoint]; bFound {
		return endpointURL, nil
	}

	//but ,if we have endpoint like event、analytic
	//and sametime the Targetkey like : events、analytics
	//we use fuzzy way to find the best match url
	var endpointURL string
	for targetKey := range dev.endpoints {
		if strings.Contains(targetKey, endpoint) {
			endpointURL = dev.endpoints[targetKey]
			return endpointURL, nil
		}
	}
	return endpointURL, errors.New("target endpoint service not found")
}

//CallMethod functions call an method, defined <method> struct.
//You should use Authenticate method to call authorized requests.
func (dev *Device) CallMethod(method interface{}, headerFileds map[string]string) (*http.Response, error) {
	pkgPath := strings.Split(reflect.TypeOf(method).PkgPath(), "/")
	pkg := strings.ToLower(pkgPath[len(pkgPath)-1])

	endpoint, err := dev.getEndpoint(pkg)
	if err != nil {
		return nil, err
	}
	return dev.callMethodDo(endpoint, method, headerFileds)
}

//CallMethod functions call an method, defined <method> struct with authentication data
func (dev *Device) callMethodDo(endpoint string, method interface{}, headerFileds map[string]string) (*http.Response, error) {
	/*
		Converting <method> struct to xml string representation
	*/
	output, err := xml.MarshalIndent(method, "  ", "    ")
	if err != nil {
		//log.Printf("error: %v\n", err.Error())
		return nil, err
	}
	//fmt.Println(gosoap.SoapMessage(string(output)).StringIndent())
	/*
		Build an SOAP request with <method>
	*/
	soap, err := buildMethodSOAP(string(output))
	if err != nil {
		//log.Printf("error: %v\n", err.Error())
		return nil, err
	}

	//fmt.Println(soap.StringIndent())
	/*
		Adding namespaces and WS-Security headers
	*/
	soap.AddRootNamespaces(Xlmns)

	//fmt.Println(soap.StringIndent())
	//Header handling for action
	// this is not a must
	//soap.AddAction()
	//Auth Handling
	if dev.login != "" && dev.password != "" {
		soap.AddWSSecurity(dev.login, dev.password)
	}

	if headerFileds != nil {
		soap.AddHeadFileds(headerFileds)
	}

	//fmt.Println(soap.StringIndent())
	/*
		Sending request and returns the response
	*/
	return networking.SendSoap(endpoint, soap.String())
}

//GetXaddr GetXaddr
func (dev *Device) GetXaddr() string {
	return dev.xaddr
}

//GetIPAddress GetIpAddress
func (dev *Device) GetIPAddress() string {
	return dev.ipaddress
}

//GetPort GetIpAddress
func (dev *Device) GetPort() int {
	return dev.port
}

//GetUser GetUserAuth
func (dev *Device) GetUser() string {

	return dev.login
}

//GetPassword GetPassword
func (dev *Device) GetPassword() string {
	return dev.password
}
