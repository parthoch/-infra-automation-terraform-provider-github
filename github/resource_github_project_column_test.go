package github

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/go-github/v18/github"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccGithubProjectColumn_basic(t *testing.T) {
	var column github.ProjectColumn

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGithubProjectColumnDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubProjectColumnConfig("new column name"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubProjectColumnExists("github_project_column.column", &column),
					testAccCheckGithubProjectColumnAttributes(&column, &testAccGithubProjectColumnExpectedAttributes{
						Name: "new column name",
					}),
				),
			},
			{
				Config: testAccGithubProjectColumnConfig("updated column name"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubProjectColumnExists("github_project_column.column", &column),
					testAccCheckGithubProjectColumnAttributes(&column, &testAccGithubProjectColumnExpectedAttributes{
						Name: "updated column name",
					}),
				),
			},
		},
	})
}

func TestAccGithubProjectColumn_importBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGithubProjectColumnDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubProjectColumnConfig("a column"),
			},
			{
				ResourceName:      "github_project_column.column",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGithubProjectColumnDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*Organization).client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_project_column" {
			continue
		}

		columnID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}

		column, res, err := conn.Projects.GetProjectColumn(context.TODO(), columnID)
		if err == nil {
			if column != nil &&
				column.GetID() == columnID {
				return fmt.Errorf("Project column still exists")
			}
		}
		if res.StatusCode != 404 {
			return err
		}
	}
	return nil
}

func testAccCheckGithubProjectColumnExists(n string, project *github.ProjectColumn) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		columnID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}

		conn := testAccProvider.Meta().(*Organization).client
		gotColumn, _, err := conn.Projects.GetProjectColumn(context.TODO(), columnID)
		if err != nil {
			return err
		}
		*project = *gotColumn
		return nil
	}
}

type testAccGithubProjectColumnExpectedAttributes struct {
	Name string
}

func testAccCheckGithubProjectColumnAttributes(column *github.ProjectColumn, want *testAccGithubProjectColumnExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if *column.Name != want.Name {
			return fmt.Errorf("got project column %q; want %q", *column.Name, want.Name)
		}

		return nil
	}
}

func testAccGithubProjectColumnConfig(columnName string) string {
	return fmt.Sprintf(`
resource "github_organization_project" "test" {
  name = "test-project"
  body = "this is a test project"
}

resource "github_project_column" "column" {
  project_id = "${github_organization_project.test.id}"
  name       = "%s"
}
`, columnName)
}
