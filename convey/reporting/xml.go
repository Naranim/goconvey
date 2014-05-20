package reporting

import (
	"encoding/xml"
	"sync"
	"sync/atomic"
)

var (
	initialized bool
	mutex       sync.Mutex
	toWrite     chan toWriteRequest
	activeTests int32
)

type toWriteRequest struct {
	suite    *xmlTestSuite
	response chan bool
}

type XmlReporter struct {
	testSuite *xmlTestSuite
	toWrite   chan AssertionResult
}

type xmlTestSuites struct {
	XMLName   xml.Name `xml:"testsuites"`
	Tests     int      `xml:"tests,attr"`
	Failures  int      `xml:"failures,attr"`
	Errors    int      `xml:"errors,attr"`
	Skipped   int      `xml:"skipped,attr"`
	TestSuite []*xmlTestSuite
}

type xmlTestSuite struct {
	XMLName   xml.Name `xml:"testsuite"`
	Name      string   `xml:"name,attr"`
	Tests     int      `xml:"tests,attr"`
	Failures  int      `xml:"failures,attr"`
	Errors    int      `xml:"errors,attr"`
	Skipped   int      `xml:"skipped,attr"`
	TestCases []*xmlTestCase
}

type xmlTestCase struct {
	XMLName        xml.Name `xml:"testcase,omitempty"`
	Skipped        bool     `xml:"skipped,omitempty"`
	FailureMessage string   `xml:"failure,omitempty>message,omitempty"`
}

func (self *XmlReporter) BeginStory(story *StoryReport) {
	atomic.AddInt32(&activeTests, 1)
}

func (self *XmlReporter) Enter(scope *ScopeReport) {
	self.testSuite = suiteFromScope(scope)
}

func (self *XmlReporter) Report(r *AssertionResult) {
	switch {
	case r.Error != nil:
		self.testSuite.Errors++
	case r.Failure != "":
		self.testSuite.Failures++
	case r.Skipped:
		self.testSuite.Skipped++
	}
	self.testSuite.Tests++

	self.testSuite.TestCases = append(self.testSuite.TestCases, caseFromResult(r))
}

func (self *XmlReporter) Exit() {
}

func (self *XmlReporter) EndStory() {
	self.report()
	self.refresh()
}

func (self *XmlReporter) report() {
	response := make(chan bool)
	toWrite <- toWriteRequest{self.testSuite, response}
	<-response
}

func (self *XmlReporter) refresh() {
	self.testSuite = newTestSuite()
}

func (self *xmlTestSuites) addSuite(suite *xmlTestSuite) {
	self.Errors += suite.Errors
	self.Failures += suite.Failures
	self.Skipped += suite.Skipped
	self.Tests += suite.Tests
	self.TestSuite = append(self.TestSuite, suite)
}

func (self *xmlTestSuites) Print(file *Printer) {
	buffer, err := xml.MarshalIndent(self, indentPrefix, indent)
	if err != nil {
		panic(err)
	}
	_, err = file.out.Seek(0, 0)
	if err != nil {
		panic(err)
	}
	lastWritten := 0
	for {
		n, err := file.out.Write(buffer[lastWritten:])
		if err != nil {
			panic(err)
		}
		lastWritten += n
		if lastWritten == len(buffer) {
			break
		}
	}
}

func writerFunction(file *Printer) {
	suites := newTestSuites()
	for {
		request := <-toWrite

		suites.addSuite(request.suite)

		atomic.AddInt32(&activeTests, -1)

		if activeTests == 0 {
			suites.Print(file)
		}

		request.response <- true
	}
}

func NewXmlReporter(file *Printer) *XmlReporter {
	mutex.Lock()
	if !initialized {
		initialized = true
		toWrite = make(chan toWriteRequest)
		go writerFunction(file)
	}
	mutex.Unlock()

	self := new(XmlReporter)
	self.testSuite = newTestSuite()
	return self
}

func suiteFromScope(scope *ScopeReport) *xmlTestSuite {
	self := new(xmlTestSuite)
	self.Name = scope.Title
	return self
}

func caseFromResult(r *AssertionResult) *xmlTestCase {
	self := new(xmlTestCase)
	self.Skipped = r.Skipped
	self.FailureMessage = r.Failure
	return self
}

func newTestSuite() *xmlTestSuite {
	self := new(xmlTestSuite)
	self.TestCases = []*xmlTestCase{}
	return self
}

func newTestSuites() *xmlTestSuites {
	self := new(xmlTestSuites)
	self.TestSuite = []*xmlTestSuite{}
	return self
}

const indentPrefix = ""
const indent = "  "

// Example output:
//
// <testsuites tests="13" failures="1" errors="0" skipped="1">
//  <testsuite name="Empty graph" tests="1" failures="0" errors="0" skipped="0">
//   <testcase></testcase>
//  </testsuite>
//  <testsuite name="Multiple insertions" tests="3" failures="0" errors="0" skipped="0">
//   <testcase></testcase>
//   <testcase></testcase>
//   <testcase></testcase>
//  </testsuite>
//  <testsuite name="Inserting edges" tests="1" failures="0" errors="0" skipped="1">
//   <testcase>
//    <skipped>true</skipped>
//   </testcase>
//  </testsuite>
//  <testsuite name="Adding zero" tests="3" failures="0" errors="0" skipped="0">
//   <testcase></testcase>
//   <testcase></testcase>
//   <testcase></testcase>
//  </testsuite>
//  <testsuite name="Adding positive" tests="3" failures="0" errors="0" skipped="0">
//   <testcase></testcase>
//   <testcase></testcase>
//   <testcase></testcase>
//  </testsuite>
//  <testsuite name="Adding negative" tests="2" failures="1" errors="0" skipped="0">
//   <testcase></testcase>
//   <testcase>
//    <failure>Expected:   -5 Actual:   -2(Should be equal)</failure>
//   </testcase>
//  </testsuite>
// </testsuites>
