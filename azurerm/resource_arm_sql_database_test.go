package azurerm

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/jen20/riviera/sql"
)

func TestResourceAzureRMSqlDatabaseEdition_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Random",
			ErrCount: 1,
		},
		{
			Value:    "Basic",
			ErrCount: 0,
		},
		{
			Value:    "Standard",
			ErrCount: 0,
		},
		{
			Value:    "Premium",
			ErrCount: 0,
		},
		{
			Value:    "DataWarehouse",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateArmSqlDatabaseEdition(tc.Value, "azurerm_sql_database")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM SQL Database edition to trigger a validation error")
		}
	}
}

func TestAccAzureRMSqlDatabase_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := testAccAzureRMSqlDatabase_basic(ri, testLocation())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSqlDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlDatabaseExists("azurerm_sql_database.test"),
				),
			},
		},
	})
}

func TestAccAzureRMSqlDatabase_elasticPool(t *testing.T) {
	ri := acctest.RandInt()
	config := testAccAzureRMSqlDatabase_elasticPool(ri, testLocation())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSqlDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlDatabaseExists("azurerm_sql_database.test"),
					resource.TestCheckResourceAttr("azurerm_sql_database.test", "elastic_pool_name", fmt.Sprintf("acctestep%d", ri)),
				),
			},
		},
	})
}

func TestAccAzureRMSqlDatabase_withTags(t *testing.T) {
	ri := acctest.RandInt()
	location := testLocation()
	preConfig := testAccAzureRMSqlDatabase_withTags(ri, location)
	postConfig := testAccAzureRMSqlDatabase_withTagsUpdate(ri, location)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSqlDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlDatabaseExists("azurerm_sql_database.test"),
					resource.TestCheckResourceAttr(
						"azurerm_sql_database.test", "tags.%", "2"),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlDatabaseExists("azurerm_sql_database.test"),
					resource.TestCheckResourceAttr(
						"azurerm_sql_database.test", "tags.%", "1"),
				),
			},
		},
	})
}

func TestAccAzureRMSqlDatabase_dataWarehouse(t *testing.T) {
	ri := acctest.RandInt()
	config := testAccAzureRMSqlDatabase_dataWarehouse(ri, testLocation())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSqlDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlDatabaseExists("azurerm_sql_database.test"),
				),
			},
		},
	})
}

func TestAccAzureRMSqlDatabase_restorePointInTime(t *testing.T) {
	ri := acctest.RandInt()
	location := testLocation()
	preConfig := testAccAzureRMSqlDatabase_basic(ri, location)
	timeToRestore := time.Now().Add(15 * time.Minute)
	formattedTime := string(timeToRestore.UTC().Format(time.RFC3339))
	postCongif := testAccAzureRMSqlDatabase_restorePointInTime(ri, formattedTime, testLocation())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSqlDatabaseDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				PreventPostDestroyRefresh: true,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlDatabaseExists("azurerm_sql_database.test"),
				),
			},
			{
				PreConfig: func() { time.Sleep(timeToRestore.Sub(time.Now().Add(-1 * time.Minute))) },
				Config:    postCongif,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlDatabaseExists("azurerm_sql_database.test"),
					testCheckAzureRMSqlDatabaseExists("azurerm_sql_database.test_restore"),
				),
			},
		},
	})
}

func testCheckAzureRMSqlDatabaseExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).rivieraClient

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &sql.GetDatabase{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetDatabase: %s", err)
		}
		if !readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: GetDatabase: %s", readResponse.Error)
		}

		return nil
	}
}

func testCheckAzureRMSqlDatabaseDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).rivieraClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_sql_database" {
			continue
		}

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &sql.GetDatabase{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetDatabase: %s", err)
		}

		if readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: SQL Database still exists: %s", readResponse.Error)
		}
	}

	return nil
}

func testAccAzureRMSqlDatabase_basic(rInt int, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "%s"
}

resource "azurerm_sql_server" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "${azurerm_resource_group.test.location}"
    version = "12.0"
    administrator_login = "mradministrator"
    administrator_login_password = "thisIsDog11"
}

resource "azurerm_sql_database" "test" {
    name = "acctestdb%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    location = "${azurerm_resource_group.test.location}"
    edition = "Standard"
    collation = "SQL_Latin1_General_CP1_CI_AS"
    max_size_bytes = "1073741824"
    requested_service_objective_name = "S0"
}
`, rInt, location, rInt, rInt)
}

func testAccAzureRMSqlDatabase_withTags(rInt int, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "%s"
}

resource "azurerm_sql_server" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "${azurerm_resource_group.test.location}"
    version = "12.0"
    administrator_login = "mradministrator"
    administrator_login_password = "thisIsDog11"
}

resource "azurerm_sql_database" "test" {
    name = "acctestdb%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    location = "${azurerm_resource_group.test.location}"
    edition = "Standard"
    collation = "SQL_Latin1_General_CP1_CI_AS"
    max_size_bytes = "1073741824"
    requested_service_objective_name = "S0"

    tags {
    	environment = "staging"
    	database = "test"
    }
}
`, rInt, location, rInt, rInt)
}

func testAccAzureRMSqlDatabase_withTagsUpdate(rInt int, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "%s"
}

resource "azurerm_sql_server" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "${azurerm_resource_group.test.location}"
    version = "12.0"
    administrator_login = "mradministrator"
    administrator_login_password = "thisIsDog11"
}

resource "azurerm_sql_database" "test" {
    name = "acctestdb%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    location = "${azurerm_resource_group.test.location}"
    edition = "Standard"
    collation = "SQL_Latin1_General_CP1_CI_AS"
    max_size_bytes = "1073741824"
    requested_service_objective_name = "S0"

    tags {
    	environment = "production"
    }
}
`, rInt, location, rInt, rInt)
}

func testAccAzureRMSqlDatabase_dataWarehouse(rInt int, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "%s"
}

resource "azurerm_sql_server" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "${azurerm_resource_group.test.location}"
    version = "12.0"
    administrator_login = "mradministrator"
    administrator_login_password = "thisIsDog11"
}

resource "azurerm_sql_database" "test" {
    name = "acctestdb%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    location = "${azurerm_resource_group.test.location}"
    edition = "DataWarehouse"
    collation = "SQL_Latin1_General_CP1_CI_AS"
    requested_service_objective_name = "DW400"
}
`, rInt, location, rInt, rInt)
}

func testAccAzureRMSqlDatabase_restorePointInTime(rInt int, formattedTime string, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "%s"
}

resource "azurerm_sql_server" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "${azurerm_resource_group.test.location}"
    version = "12.0"
    administrator_login = "mradministrator"
    administrator_login_password = "thisIsDog11"
}

resource "azurerm_sql_database" "test" {
    name = "acctestdb%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    location = "${azurerm_resource_group.test.location}"
    edition = "Standard"
    collation = "SQL_Latin1_General_CP1_CI_AS"
    max_size_bytes = "1073741824"
    requested_service_objective_name = "S0"
}

resource "azurerm_sql_database" "test_restore" {
    name = "acctestdb_restore%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    location = "${azurerm_resource_group.test.location}"
    create_mode = "PointInTimeRestore"
    source_database_id = "${azurerm_sql_database.test.id}"
    restore_point_in_time = "%s"
}
`, rInt, location, rInt, rInt, rInt, formattedTime)
}

func testAccAzureRMSqlDatabase_elasticPool(rInt int, location string) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG_%d"
    location = "%s"
}

resource "azurerm_sql_server" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "${azurerm_resource_group.test.location}"
    version = "12.0"
    administrator_login = "mradministrator"
    administrator_login_password = "thisIsDog11"
}

resource "azurerm_sql_elasticpool" "test" {
    name = "acctestep%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "${azurerm_resource_group.test.location}"
    server_name = "${azurerm_sql_server.test.name}"
    edition = "Basic"
    dtu = 50
    pool_size = 5000
}

resource "azurerm_sql_database" "test" {
    name = "acctestdb%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    location = "${azurerm_resource_group.test.location}"
    edition = "${azurerm_sql_elasticpool.test.edition}"
    collation = "SQL_Latin1_General_CP1_CI_AS"
    max_size_bytes = "1073741824"
    elastic_pool_name = "${azurerm_sql_elasticpool.test.name}"
    requested_service_objective_name = "ElasticPool"
}
`, rInt, location, rInt, rInt, rInt)
}
