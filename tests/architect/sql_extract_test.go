package architect_test

import (
	"context"
	"testing"

	"sdp_dev/internal/architect"
	"sdp_dev/internal/architect/extract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLExtractor_CreateTable(t *testing.T) {
	root := setupTree(t, map[string]string{
		"schema.sql": `
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    age INTEGER
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    body TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.SQLAnalysis)

	tables := frag.SQLAnalysis.Tables
	require.Len(t, tables, 2)

	// users table
	u := tables[0]
	assert.Equal(t, "users", u.Name)
	require.Len(t, u.Columns, 3)
	assert.Equal(t, "id", u.Columns[0].Name)
	assert.True(t, u.Columns[0].PrimaryKey)
	assert.True(t, u.Columns[0].NotNull) // PK implies NOT NULL
	assert.Equal(t, "name", u.Columns[1].Name)
	assert.True(t, u.Columns[1].NotNull)
	assert.Equal(t, "age", u.Columns[2].Name)
	assert.False(t, u.Columns[2].NotNull)

	// posts table
	p := tables[1]
	assert.Equal(t, "posts", p.Name)
	require.Len(t, p.Columns, 4)
	assert.True(t, p.Columns[0].PrimaryKey)
}

func TestSQLExtractor_ForeignKeys(t *testing.T) {
	root := setupTree(t, map[string]string{
		"fk.sql": `
CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    FOREIGN KEY (customer_id) REFERENCES customers(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	fks := frag.SQLAnalysis.ForeignKeys
	require.Len(t, fks, 2)

	assert.Equal(t, "orders", fks[0].FromTable)
	assert.Equal(t, "customer_id", fks[0].FromColumn)
	assert.Equal(t, "customers", fks[0].ToTable)
	assert.Equal(t, "id", fks[0].ToColumn)

	assert.Equal(t, "orders", fks[1].FromTable)
	assert.Equal(t, "product_id", fks[1].FromColumn)
	assert.Equal(t, "products", fks[1].ToTable)
	assert.Equal(t, "id", fks[1].ToColumn)
}

func TestSQLExtractor_MigrationDetection(t *testing.T) {
	root := setupTree(t, map[string]string{
		"db/migrate/001_create_users.sql":  "CREATE TABLE users (id INTEGER);",
		"db/migrate/002_add_email.sql":     "ALTER TABLE users ADD email TEXT;",
		"db/migrate/003_create_orders.sql": "CREATE TABLE orders (id INTEGER);",
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	mig := frag.SQLAnalysis.Migrations
	require.NotNil(t, mig)
	assert.Equal(t, "db/migrate", mig.Dir)
	assert.Equal(t, 3, mig.Count)
	assert.Equal(t, "003_create_orders.sql", mig.Latest)
}

func TestSQLExtractor_PIIDetection(t *testing.T) {
	root := setupTree(t, map[string]string{
		"schema.sql": `
CREATE TABLE customers (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL,
    phone TEXT,
    ssn TEXT,
    birth_date TEXT,
    first_name TEXT,
    last_name TEXT,
    user_email TEXT,
    ip_address TEXT,
    credit_card TEXT,
    address TEXT,
    status INTEGER
);
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	pii := frag.SQLAnalysis.PIIColumns
	require.NotEmpty(t, pii)

	// Collect results by column.
	byCol := make(map[string]float64)
	for _, p := range pii {
		byCol[p.Column] = p.Confidence
	}

	// Exact matches should have 0.95 confidence.
	assert.Equal(t, 0.95, byCol["email"])
	assert.Equal(t, 0.95, byCol["phone"])
	assert.Equal(t, 0.95, byCol["ssn"])
	assert.Equal(t, 0.95, byCol["birth_date"])
	assert.Equal(t, 0.95, byCol["first_name"])
	assert.Equal(t, 0.95, byCol["last_name"])
	assert.Equal(t, 0.95, byCol["ip_address"])
	assert.Equal(t, 0.95, byCol["credit_card"])
	assert.Equal(t, 0.95, byCol["address"])

	// Partial match.
	assert.Equal(t, 0.75, byCol["user_email"])

	// Non-PII columns should not appear.
	_, found := byCol["status"]
	assert.False(t, found)
	_, found = byCol["id"]
	assert.False(t, found)
}

func TestSQLExtractor_PrismaSchema(t *testing.T) {
	root := setupTree(t, map[string]string{
		"prisma/schema.prisma": `
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id    Int     @id @default(autoincrement())
  email String  @unique
  posts Post[]
}

model Post {
  id       Int    @id @default(autoincrement())
  title    String
  authorId Int
  author   User   @relation(fields: [authorId], references: [id])
}
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	orms := frag.SQLAnalysis.ORMModels
	require.Len(t, orms, 2)

	assert.Equal(t, "prisma", orms[0].Framework)
	assert.Equal(t, "User", orms[0].Model)
	assert.Equal(t, "prisma", orms[1].Framework)
	assert.Equal(t, "Post", orms[1].Model)
}

func TestSQLExtractor_DataDomainClustering(t *testing.T) {
	root := setupTree(t, map[string]string{
		"schema.sql": `
CREATE TABLE auth_users (
    id INTEGER PRIMARY KEY
);

CREATE TABLE auth_roles (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    FOREIGN KEY (user_id) REFERENCES auth_users(id)
);

CREATE TABLE billing_invoices (
    id INTEGER PRIMARY KEY
);

CREATE TABLE billing_payments (
    id INTEGER PRIMARY KEY,
    invoice_id INTEGER,
    FOREIGN KEY (invoice_id) REFERENCES billing_invoices(id)
);

CREATE TABLE logs (
    id INTEGER PRIMARY KEY
);
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	domains := frag.SQLAnalysis.DataDomains
	require.Len(t, domains, 3, "expected 3 connected components")

	// Build a map from domain name to table list for easier assertion.
	byName := make(map[string][]string)
	for _, d := range domains {
		byName[d.Name] = d.Tables
	}

	// Auth domain: auth_users + auth_roles (connected by FK).
	authDomain := findDomainContaining(domains, "auth_users")
	require.NotNil(t, authDomain)
	assert.Len(t, authDomain.Tables, 2)
	assert.Contains(t, authDomain.Tables, "auth_users")
	assert.Contains(t, authDomain.Tables, "auth_roles")

	// Billing domain: billing_invoices + billing_payments.
	billingDomain := findDomainContaining(domains, "billing_invoices")
	require.NotNil(t, billingDomain)
	assert.Len(t, billingDomain.Tables, 2)
	assert.Contains(t, billingDomain.Tables, "billing_invoices")
	assert.Contains(t, billingDomain.Tables, "billing_payments")

	// Logs domain: standalone.
	logsDomain := findDomainContaining(domains, "logs")
	require.NotNil(t, logsDomain)
	assert.Len(t, logsDomain.Tables, 1)
}

func TestSQLExtractor_NoSQLFiles(t *testing.T) {
	root := setupTree(t, map[string]string{
		"readme.md": "# Hello",
		"main.go":   "package main",
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.SQLAnalysis)

	// No tables, no FKs, no migrations, no PII, no domains.
	assert.Empty(t, frag.SQLAnalysis.Tables)
	assert.Empty(t, frag.SQLAnalysis.ForeignKeys)
	assert.Nil(t, frag.SQLAnalysis.Migrations)
	assert.Empty(t, frag.SQLAnalysis.PIIColumns)
	assert.Empty(t, frag.SQLAnalysis.DataDomains)
}

// findDomainContaining returns the first domain whose Tables slice contains tableName.
func findDomainContaining(domains []architect.DataDomain, tableName string) *architect.DataDomain {
	for i := range domains {
		for _, t := range domains[i].Tables {
			if t == tableName {
				return &domains[i]
			}
		}
	}
	return nil
}
