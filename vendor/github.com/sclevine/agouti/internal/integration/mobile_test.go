package integration_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

func testMobile(browserName string, newPage pageFunc) {
	Describe("mobile test for "+browserName, func() {
		var (
			page   *agouti.Page
			server *httptest.Server
		)

		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				html, _ := ioutil.ReadFile("mobile_test_page.html")
				response.Write(html)
			}))

			var err error
			page, err = newPage()
			Expect(err).NotTo(HaveOccurred())

			Expect(page.Size(640, 480)).To(Succeed())
			port := strings.Split(server.URL, ":")[2]
			Expect(page.Navigate("http://10.0.2.2:" + port)).To(Succeed())
		})

		AfterEach(func() {
			Expect(page.Destroy()).To(Succeed())
			server.Close()
		})

		It("should support various touch events", func() {
			touch := page.Find("#touch")
			message := page.Find("#message")

			By("performing tap actions", func() {
				Expect(touch.Tap(agouti.SingleTap)).To(Succeed())
				Eventually(message).Should(HaveText("event: start with 1 end with 0"))
				Expect(page.Refresh()).To(Succeed())
				Expect(touch.Tap(agouti.DoubleTap)).To(Succeed())
				Eventually(message).Should(HaveText("event: start with 1 end with 0 start with 1 end with 0"))
			})

			By("performing touch actions", func() {
				Expect(page.Refresh()).To(Succeed())
				Expect(touch.Touch(agouti.HoldFinger)).To(Succeed())
				Eventually(message).Should(HaveText("event: start with 1"))
				Expect(page.Refresh()).To(Succeed())
				Expect(touch.Touch(agouti.HoldFinger)).To(Succeed())
				Expect(touch.Touch(agouti.ReleaseFinger)).To(Succeed())
				Eventually(message).Should(HaveText("event: start with 1 end with 0"))
			})

			By("performing a flick", func() {
				Expect(page.Refresh()).To(Succeed())
				Expect(touch.FlickFinger(10, 10, 10)).To(Succeed())
				Eventually(message).Should(HaveText("event: start with 1 end with 0"))
			})

			By("performing a finger scroll", func() {
				Expect(page.Refresh()).To(Succeed())
				Expect(touch.ScrollFinger(40, 50)).To(Succeed())
				Eventually(message).Should(MatchText("event: scroll left [45][0-9] scroll top [56][0-9]"))
			})
		})
	})
}
