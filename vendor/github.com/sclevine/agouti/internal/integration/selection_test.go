package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

func testSelection(browserName string, newPage pageFunc) {
	Describe("selection test for "+browserName, func() {
		var (
			page      *agouti.Page
			server    *httptest.Server
			submitted bool
		)

		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.Method == "POST" {
					submitted = true
				}
				html, _ := ioutil.ReadFile("test_page.html")
				response.Write(html)
			}))

			var err error
			page, err = newPage()
			Expect(err).NotTo(HaveOccurred())

			Expect(page.Size(640, 480)).To(Succeed())
			Expect(page.Navigate(server.URL)).To(Succeed())
		})

		AfterEach(func() {
			Expect(page.Destroy()).To(Succeed())
			server.Close()
		})

		It("should support asserting on element identity", func() {
			By("asserting on an element's existence", func() {
				Expect(page.Find("header")).To(BeFound())
				Expect(page.Find("header")).To(HaveCount(1))
				Expect(page.Find("not-a-header")).NotTo(BeFound())
			})

			By("comparing two selections for equality", func() {
				Expect(page.Find("#some_element")).To(EqualElement(page.FindByXPath("//div[@class='some-element']")))
				Expect(page.Find("#some_element")).NotTo(EqualElement(page.Find("header")))
			})
		})

		It("should support moving the mouse pointer over a selected element", func() {
			Expect(page.Find("#some_checkbox").MouseToElement()).To(Succeed())
			Expect(page.Click(agouti.SingleClick, agouti.LeftButton)).To(Succeed())
			Expect(page.Find("#some_checkbox")).To(BeSelected())
		})

		It("should support selecting elements", func() {
			By("finding an element by selection index", func() {
				Expect(page.All("option").At(0)).To(HaveText("first option"))
				Expect(page.All("select").At(1).First("option")).To(HaveText("third option"))
			})

			By("finding an element by chained selectors", func() {
				Expect(page.Find("header").Find("h1")).To(HaveText("Title"))
				Expect(page.Find("header").FindByXPath("//h1")).To(HaveText("Title"))
			})

			By("finding an element by link text", func() {
				Expect(page.FindByLink("Click Me").Attribute("href")).To(HaveSuffix("#new_page"))
			})

			By("finding an element by label text", func() {
				Expect(page.FindByLabel("Some Label")).To(HaveAttribute("value", "some labeled value"))
				Expect(page.FindByLabel("Some Container Label")).To(HaveAttribute("value", "some embedded value"))
			})

			By("finding an element by button text", func() {
				Expect(page.FindByButton("Some Button")).To(HaveAttribute("name", "some button name"))
				Expect(page.FindByButton("Some Input Button")).To(HaveAttribute("type", "button"))
				Expect(page.FindByButton("Some Submit Button")).To(HaveAttribute("type", "submit"))
			})

			By("finding an element by name attibute", func() {
				Expect(page.FindByName("some button name")).To(HaveAttribute("name", "some button name"))
			})

			By("finding multiple elements", func() {
				Expect(page.All("select").All("option")).To(BeVisible())
				Expect(page.All("h1,h2")).NotTo(BeVisible())
			})
		})

		It("should support retrieving element properties", func() {
			By("asserting on element text", func() {
				Expect(page.Find("header")).To(HaveText("Title"))
				Expect(page.Find("header")).NotTo(HaveText("Not-Title"))
				Expect(page.Find("header")).To(MatchText("T.+e"))
				Expect(page.Find("header")).NotTo(MatchText("X.+e"))
			})

			By("asserting on whether elements are active", func() {
				Expect(page.Find("#labeled_field")).NotTo(BeActive())
				Expect(page.Find("#labeled_field").Click()).To(Succeed())
				Expect(page.Find("#labeled_field")).To(BeActive())
			})

			By("asserting on element attributes", func() {
				Expect(page.Find("#some_checkbox")).To(HaveAttribute("type", "checkbox"))
			})

			By("asserting on element CSS", func() {
				Expect(page.Find("#some_element")).To(HaveCSS("color", "rgba(0, 0, 255, 1)"))
				Expect(page.Find("#some_element")).To(HaveCSS("color", "rgb(0, 0, 255)"))
				Expect(page.Find("#some_element")).To(HaveCSS("color", "blue"))
			})

			By("asserting on whether elements are selected", func() {
				Expect(page.Find("#some_checkbox")).NotTo(BeSelected())
				Expect(page.Find("#some_selected_checkbox")).To(BeSelected())
			})

			By("asserting on element visibility", func() {
				Expect(page.Find("header h1")).To(BeVisible())
				Expect(page.Find("header h2")).NotTo(BeVisible())
			})

			By("asserting on whether elements are enabled", func() {
				Expect(page.Find("#some_checkbox")).To(BeEnabled())
				Expect(page.Find("#some_disabled_checkbox")).NotTo(BeEnabled())
			})
		})

		It("should support element actions", func() {
			By("clicking on an element", func() {
				checkbox := page.Find("#some_checkbox")
				Expect(checkbox.Click()).To(Succeed())
				Expect(checkbox).To(BeSelected())
				Expect(checkbox.Click()).To(Succeed())
				Expect(checkbox).NotTo(BeSelected())
			})

			By("double-clicking on an element", func() {
				selection := page.Find("#double_click")
				Expect(selection.DoubleClick()).To(Succeed())
				Expect(selection).To(HaveText("double-click success"))
			})

			By("filling out an element", func() {
				Expect(page.Find("#some_input").Fill("some other value")).To(Succeed())
				Expect(page.Find("#some_input")).To(HaveAttribute("value", "some other value"))
			})

			// NOTE: PhantomJS regression causes crash on file upload
			if browserName != "PhantomJS" {
				By("uploading a file", func() {
					Expect(page.Find("#file_picker").UploadFile("test_page.html")).To(Succeed())
					var result string
					Expect(page.RunScript("return document.getElementById('file_picker').value;", nil, &result)).To(Succeed())
					Expect(result).To(HaveSuffix("test_page.html"))
				})
			}

			By("checking and unchecking a checkbox", func() {
				checkbox := page.Find("#some_checkbox")
				Expect(checkbox.Uncheck()).To(Succeed())
				Expect(checkbox).NotTo(BeSelected())
				Expect(checkbox.Check()).To(Succeed())
				Expect(checkbox).To(BeSelected())
				Expect(checkbox.Uncheck()).To(Succeed())
				Expect(checkbox).NotTo(BeSelected())
			})

			By("selecting an option by text", func() {
				selection := page.Find("#some_select")
				Expect(selection.All("option").At(1)).NotTo(BeSelected())
				Expect(selection.Select("second option")).To(Succeed())
				Expect(selection.All("option").At(1)).To(BeSelected())
			})

			By("submitting a form", func() {
				Expect(page.Find("#some_form").Submit()).To(Succeed())
				Eventually(func() bool { return submitted }).Should(BeTrue())
			})
		})
	})
}
