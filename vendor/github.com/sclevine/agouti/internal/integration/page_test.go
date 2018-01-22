package integration_test

import (
	"image/png"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

func testPage(browserName string, newPage pageFunc) {
	Describe("page test for "+browserName, func() {
		var (
			page   *agouti.Page
			server *httptest.Server
		)

		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
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

		It("should support retrieving page properties", func() {
			Expect(page).To(HaveTitle("Page Title"))
			Expect(page).To(HaveURL(server.URL + "/"))
			Expect(page.HTML()).To(ContainSubstring("<h1>Title</h1>"))
		})

		It("should support JavaScript", func() {
			By("waiting for page JavaScript to take effect", func() {
				Expect(page.Find("#some_element")).NotTo(HaveText("some text"))
				Eventually(page.Find("#some_element"), "4s").Should(HaveText("some text"))
				Consistently(page.Find("#some_element")).Should(HaveText("some text"))
			})

			// NOTE: disabled due to recent Firefox regression with passing args
			if browserName != "Firefox" {
				By("executing provided JavaScript", func() {
					arguments := map[string]interface{}{"elementID": "some_element"}
					var result string
					Expect(page.RunScript("return document.getElementById(elementID).innerHTML;", arguments, &result)).To(Succeed())
					Expect(result).To(Equal("some text"))
				})
			}
		})

		It("should support taking screenshots", func() {
			Expect(page.Screenshot(".test.screenshot.png")).To(Succeed())
			defer os.Remove(".test.screenshot.png")
			file, _ := os.Open(".test.screenshot.png")
			_, err := png.Decode(file)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should support links and navigation", func() {
			By("clicking on a link", func() {
				Expect(page.FindByLink("Click Me").Click()).To(Succeed())
				Expect(page.URL()).To(ContainSubstring("#new_page"))
			})

			By("navigating through browser history", func() {
				Expect(page.Back()).To(Succeed())
				Expect(page.URL()).NotTo(ContainSubstring("#new_page"))
				Expect(page.Forward()).To(Succeed())
				Expect(page.URL()).To(ContainSubstring("#new_page"))
			})

			By("refreshing the page", func() {
				checkbox := page.Find("#some_checkbox")
				Expect(checkbox.Check()).To(Succeed())
				Expect(page.Refresh()).To(Succeed())
				Expect(checkbox).NotTo(BeSelected())
			})
		})

		// NOTE: browsers besides PhantomJS do not support JavaScript logs
		if browserName == "PhantomJS" {
			It("should support retrieving logs", func() {
				Eventually(page).Should(HaveLoggedInfo("some log"))
				Expect(page).NotTo(HaveLoggedError())
				Eventually(page, "4s").Should(HaveLoggedError("ReferenceError: Can't find variable: doesNotExist\n  (anonymous function)"))
			})
		}

		It("should support switching frames", func() {
			By("switching to an iframe", func() {
				Expect(page.Find("#frame").SwitchToFrame()).To(Succeed())
				Expect(page.Find("body")).To(MatchText("Example Domain"))
			})

			// NOTE: PhantomJS does not support Page.SwitchToParentFrame
			if browserName != "PhantomJS" {
				By("switching back to the default frame by referring to the parent frame", func() {
					Expect(page.SwitchToParentFrame()).To(Succeed())
					Expect(page.Find("body")).NotTo(MatchText("Example Domain"))
				})

				Expect(page.Find("#frame").SwitchToFrame()).To(Succeed())
			}

			By("switching back to the default frame by referring to the root frame", func() {
				Expect(page.SwitchToRootFrame()).To(Succeed())
				Expect(page.Find("body")).NotTo(MatchText("Example Domain"))
			})
		})

		It("should support switching windows", func() {
			Expect(page.Find("#new_window").Click()).To(Succeed())
			Expect(page).To(HaveWindowCount(2))

			By("switching windows", func() {
				Expect(page.SwitchToWindow("new window")).To(Succeed())
				Expect(page.Find("header")).NotTo(BeFound())
				Expect(page.NextWindow()).To(Succeed())
				Expect(page.Find("header")).To(BeFound())
			})

			By("closing windows", func() {
				Expect(page.CloseWindow()).To(Succeed())
				Expect(page).To(HaveWindowCount(1))
			})
		})

		// NOTE: PhantomJS does not support popup boxes
		if browserName != "PhantomJS" {
			It("should support popup boxes", func() {
				By("interacting with alert popups", func() {
					Expect(page.Find("#popup_alert").Click()).To(Succeed())
					Expect(page).To(HavePopupText("some alert"))
					Expect(page.ConfirmPopup()).To(Succeed())
				})

				By("interacting with confirm boxes", func() {
					var confirmed bool

					Expect(page.Find("#popup_confirm").Click()).To(Succeed())

					Expect(page.ConfirmPopup()).To(Succeed())
					Expect(page.RunScript("return confirmed;", nil, &confirmed)).To(Succeed())
					Expect(confirmed).To(BeTrue())

					Expect(page.Find("#popup_confirm").Click()).To(Succeed())

					Expect(page.CancelPopup()).To(Succeed())
					Expect(page.RunScript("return confirmed;", nil, &confirmed)).To(Succeed())
					Expect(confirmed).To(BeFalse())
				})

				By("interacting with prompt boxes", func() {
					var promptText string

					Expect(page.Find("#popup_prompt").Click()).To(Succeed())

					Expect(page.EnterPopupText("banana")).To(Succeed())
					Expect(page.ConfirmPopup()).To(Succeed())
					Expect(page.RunScript("return promptText;", nil, &promptText)).To(Succeed())
					Expect(promptText).To(Equal("banana"))
				})
			})
		}

		It("should support manipulating and retrieving cookies", func() {
			Expect(page.SetCookie(&http.Cookie{Name: "webdriver-test-cookie", Value: "webdriver value"})).To(Succeed())
			cookies, err := page.GetCookies()
			Expect(err).NotTo(HaveOccurred())
			cookieNames := []string{cookies[0].Name, cookies[1].Name}
			Expect(cookieNames).To(ConsistOf("webdriver-test-cookie", "browser-test-cookie"))
			Expect(page.DeleteCookie("browser-test-cookie")).To(Succeed())
			Expect(page.GetCookies()).To(HaveLen(1))
			Expect(page.ClearCookies()).To(Succeed())
			Expect(page.GetCookies()).To(HaveLen(0))
		})

		It("should support resetting the page", func() {
			Expect(page.SetCookie(&http.Cookie{Name: "webdriver-test-cookie", Value: "webdriver value"})).To(Succeed())
			Expect(page.GetCookies()).To(HaveLen(2))

			Expect(page.RunScript("localStorage.setItem('some-local-storage-key', 'some-local-storage-value');", nil, nil)).To(Succeed())
			var localStorageTest string
			Expect(page.RunScript("return localStorage.getItem('some-local-storage-key')", nil, &localStorageTest)).To(Succeed())
			Expect(localStorageTest).To(Equal("some-local-storage-value"))

			Expect(page.RunScript("sessionStorage.setItem('some-session-storage-key', 'some-session-storage-value');", nil, nil)).To(Succeed())
			var sessionStorageTest string
			Expect(page.RunScript("return sessionStorage.getItem('some-session-storage-key')", nil, &sessionStorageTest)).To(Succeed())
			Expect(sessionStorageTest).To(Equal("some-session-storage-value"))

			Expect(page.Find("#popup_alert").Click()).To(Succeed())

			Expect(page.Reset()).To(Succeed())

			By("navigating to about:blank", func() {
				Expect(page.URL()).To(Equal("about:blank"))
			})

			Expect(page.Navigate(server.URL)).To(Succeed())

			By("deleting all cookies for the current domain", func() {
				Expect(page.GetCookies()).To(HaveLen(1))
			})

			By("deleting local storage for the current domain", func() {
				var localStorageTest string
				Expect(page.RunScript("return localStorage.getItem('some-local-storage-key');", nil, &localStorageTest)).To(Succeed())
				Expect(localStorageTest).To(BeEmpty())
			})

			By("deleting session storage for the current domain", func() {
				var sessionStorageTest string
				Expect(page.RunScript("return sessionStorage.getItem('some-session-storage-key');", nil, &sessionStorageTest)).To(Succeed())
				Expect(sessionStorageTest).To(BeEmpty())
			})

			By("allowing reset to be called multiple times", func() {
				Expect(page.Reset()).To(Succeed())
				Expect(page.Reset()).To(Succeed())
				Expect(page.Navigate(server.URL)).To(Succeed())
			})
		})

		It("should support various mouse events", func() {
			checkbox := page.Find("#some_checkbox")

			By("moving from the disabled checkbox a regular checkbox", func() {
				disabledCheckbox := page.Find("#some_disabled_checkbox")
				Expect(disabledCheckbox.MouseToElement())
				Expect(page.MoveMouseBy(-24, 0)).To(Succeed())

				// NOTE: Firefox does not move the mouse by an offset correctly
				if browserName == "Firefox" {
					Expect(checkbox.MouseToElement())
				}
			})

			By("single clicking on a checkbox", func() {
				Expect(page.Click(agouti.SingleClick, agouti.LeftButton)).To(Succeed())
				Expect(checkbox).To(BeSelected())
			})

			By("holding and releasing a click on a checkbox", func() {
				Expect(page.Click(agouti.HoldClick, agouti.LeftButton)).To(Succeed())
				Expect(page.Click(agouti.ReleaseClick, agouti.LeftButton)).To(Succeed())
				Expect(checkbox).NotTo(BeSelected())
			})

			By("moving the mouse pointer and double clicking", func() {
				doubleClick := page.Find("#double_click")
				Expect(doubleClick.MouseToElement()).To(Succeed())
				Expect(page.DoubleClick()).To(Succeed())
				Expect(doubleClick).To(HaveText("double-click success"))
			})
		})
	})
}
