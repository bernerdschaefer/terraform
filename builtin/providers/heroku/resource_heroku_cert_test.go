package heroku

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHerokuCert_Basic(t *testing.T) {
	var endpoint heroku.SSLEndpointInfoResult
	appName := fmt.Sprintf("tftest-%s", acctest.RandString(10))

	wd, _ := os.Getwd()
	certFile := wd + "/test-fixtures/terraform.cert"
	keyFile := wd + "/test-fixtures/terraform.key"
	keyFile2 := wd + "/test-fixtures/terraform2.key"

	certificateChainBytes, _ := ioutil.ReadFile(certFile)
	certificateChain := string(certificateChainBytes)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuCertDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckHerokuCertConfig(appName, certFile, keyFile),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuCertExists("heroku_cert.ssl_certificate", &endpoint),
					testAccCheckHerokuCertificateChain(&endpoint, certificateChain),
					resource.TestCheckResourceAttr(
						"heroku_cert.ssl_certificate",
						"cname", fmt.Sprintf("%s.herokuapp.com", appName)),
				),
			},
			{
				Config: testAccCheckHerokuCertConfig(appName, certFile, keyFile2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuCertExists("heroku_cert.ssl_certificate", &endpoint),
					testAccCheckHerokuCertificateChain(&endpoint, certificateChain),
					resource.TestCheckResourceAttr(
						"heroku_cert.ssl_certificate",
						"cname", fmt.Sprintf("%s.herokuapp.com", appName)),
				),
			},
		},
	})
}

func testAccCheckHerokuCertConfig(appName, certFile, keyFile string) string {
	return fmt.Sprintf(`
resource "heroku_app" "foobar" {
  name = "%s"
  region = "eu"
}

resource "heroku_addon" "ssl" {
  app = "${heroku_app.foobar.name}"
  plan = "ssl:endpoint"
}

resource "heroku_cert" "ssl_certificate" {
  app = "${heroku_app.foobar.name}"
  depends_on = ["heroku_addon.ssl"]
  certificate_chain="${file("%s")}"
  private_key="${file("%s")}"
}`, appName, certFile, keyFile)
}

func testAccCheckHerokuCertDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*heroku.Service)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "heroku_cert" {
			continue
		}

		_, err := client.SSLEndpointInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Cerfificate still exists")
		}
	}

	return nil
}

func testAccCheckHerokuCertificateChain(endpoint *heroku.SSLEndpointInfoResult, chain string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if endpoint.CertificateChain != chain {
			return fmt.Errorf("Bad certificate chain: %s", endpoint.CertificateChain)
		}

		return nil
	}
}

func testAccCheckHerokuCertExists(n string, endpoint *heroku.SSLEndpointInfoResult) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SSL endpoint ID is set")
		}

		client := testAccProvider.Meta().(*heroku.Service)

		foundEndpoint, err := client.SSLEndpointInfo(context.TODO(), rs.Primary.Attributes["app"], rs.Primary.ID)

		if err != nil {
			return err
		}

		if foundEndpoint.ID != rs.Primary.ID {
			return fmt.Errorf("SSL endpoint not found")
		}

		*endpoint = *foundEndpoint

		return nil
	}
}
