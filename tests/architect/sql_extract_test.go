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

// ---------------------------------------------------------------------------
// New tests for WS-09 improvements
// ---------------------------------------------------------------------------

func TestSQLExtractor_SchemaQualifiedTables(t *testing.T) {
	root := setupTree(t, map[string]string{
		"schema.sql": `
CREATE TABLE public.users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE api.orders (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    FOREIGN KEY (user_id) REFERENCES public.users(id)
);
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.SQLAnalysis)

	tables := frag.SQLAnalysis.Tables
	require.Len(t, tables, 2)

	// Verify table names (without schema prefix in the Name field).
	assert.Equal(t, "users", tables[0].Name)
	assert.Equal(t, "public", tables[0].Schema)
	assert.Equal(t, "orders", tables[1].Name)
	assert.Equal(t, "api", tables[1].Schema)

	// FK should reference schema-qualified table.
	fks := frag.SQLAnalysis.ForeignKeys
	require.Len(t, fks, 1)
	assert.Equal(t, "orders", fks[0].FromTable)
	assert.Equal(t, "public.users", fks[0].ToTable)
}

func TestSQLExtractor_StoredProcedures(t *testing.T) {
	root := setupTree(t, map[string]string{
		"procs.sql": `
CREATE FUNCTION calculate_total(order_id INTEGER) RETURNS DECIMAL AS $$
BEGIN
    RETURN 0;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE PROCEDURE refresh_materialized_views()
LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW summary_view;
END;
$$;
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.SQLAnalysis)

	procs := frag.SQLAnalysis.StoredProcs
	require.Len(t, procs, 2)
	assert.Equal(t, "calculate_total", procs[0].Name)
	assert.Equal(t, "refresh_materialized_views", procs[1].Name)
}

func TestSQLExtractor_ExpandedPII(t *testing.T) {
	root := setupTree(t, map[string]string{
		"schema.sql": `
CREATE TABLE profiles (
    id INTEGER PRIMARY KEY,
    passport TEXT,
    mobile_number TEXT,
    zip_code TEXT,
    postal_code TEXT,
    maiden_name TEXT,
    date_of_birth TEXT,
    tax_id TEXT,
    driver_license TEXT,
    user_passport_number TEXT
);
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	pii := frag.SQLAnalysis.PIIColumns
	require.NotEmpty(t, pii)

	byCol := make(map[string]architect.PIIColumn)
	for _, p := range pii {
		byCol[p.Column] = p
	}

	// Exact matches at 0.95.
	testCases := []struct {
		col      string
		piiType  string
		conf     float64
	}{
		{"passport", "government_identifier", 0.95},
		{"mobile_number", "phone_number", 0.75},
		{"zip_code", "location_identifier", 0.95},
		{"postal_code", "location_identifier", 0.95},
		{"maiden_name", "personal_name", 0.95},
		{"date_of_birth", "date_of_birth", 0.95},
		{"tax_id", "government_identifier", 0.95},
		{"driver_license", "government_identifier", 0.95},
		{"user_passport_number", "government_identifier", 0.75},
	}

	for _, tc := range testCases {
		p, ok := byCol[tc.col]
		require.True(t, ok, "expected PII detection for column %q", tc.col)
		assert.Equal(t, tc.conf, p.Confidence, "confidence for %q", tc.col)
		assert.Equal(t, tc.piiType, p.PIIType, "PIIType for %q", tc.col)
	}
}

func TestSQLExtractor_PIITypeField(t *testing.T) {
	root := setupTree(t, map[string]string{
		"schema.sql": `
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL,
    phone TEXT,
    ssn TEXT,
    first_name TEXT,
    last_name TEXT,
    credit_card TEXT
);
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	byCol := make(map[string]architect.PIIColumn)
	for _, p := range frag.SQLAnalysis.PIIColumns {
		byCol[p.Column] = p
	}

	// Verify PIIType is populated correctly.
	assert.Equal(t, "email_address", byCol["email"].PIIType)
	assert.Equal(t, "phone_number", byCol["phone"].PIIType)
	assert.Equal(t, "social_security_number", byCol["ssn"].PIIType)
	assert.Equal(t, "personal_name", byCol["first_name"].PIIType)
	assert.Equal(t, "personal_name", byCol["last_name"].PIIType)
	assert.Equal(t, "financial_identifier", byCol["credit_card"].PIIType)
}

func TestSQLExtractor_GORMModelNames(t *testing.T) {
	root := setupTree(t, map[string]string{
		"models/user.go": `package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name  string
	Email string ` + "`gorm:\"column:email\"`" + `
}

type Order struct {
	gorm.Model
	Total float64
}
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	models := frag.SQLAnalysis.ORMModels
	require.NotEmpty(t, models, "expected GORM models to be detected")

	// Should detect both struct names.
	names := make(map[string]bool)
	for _, m := range models {
		assert.Equal(t, "gorm", m.Framework)
		names[m.Model] = true
	}
	assert.True(t, names["User"], "expected User model")
	assert.True(t, names["Order"], "expected Order model")
}

func TestSQLExtractor_DjangoModelNames(t *testing.T) {
	root := setupTree(t, map[string]string{
		"app/models.py": `from django.db import models

class Product(models.Model):
    name = models.CharField(max_length=100)
    price = models.DecimalField(max_digits=10, decimal_places=2)

class Category(models.Model):
    name = models.CharField(max_length=50)
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	models := frag.SQLAnalysis.ORMModels
	require.Len(t, models, 2)
	assert.Equal(t, "django", models[0].Framework)
	assert.Equal(t, "Product", models[0].Model)
	assert.Equal(t, "django", models[1].Framework)
	assert.Equal(t, "Category", models[1].Model)
}

func TestSQLExtractor_SQLAlchemyModelNames(t *testing.T) {
	root := setupTree(t, map[string]string{
		"models.py": `from sqlalchemy import Column, Integer, String
from sqlalchemy.ext.declarative import declarative_base

Base = declarative_base()

class User(Base):
    __tablename__ = "users"
    id = Column(Integer, primary_key=True)
    name = Column(String)
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	models := frag.SQLAnalysis.ORMModels
	require.NotEmpty(t, models)

	// Should find SQLAlchemy model.
	saModels := make(map[string]bool)
	for _, m := range models {
		if m.Framework == "sqlalchemy" {
			saModels[m.Model] = true
		}
	}
	assert.True(t, saModels["User"], "expected SQLAlchemy User model")
}

func TestSQLExtractor_JPAModelNames(t *testing.T) {
	root := setupTree(t, map[string]string{
		"entity/User.java": `package com.example.entity;

import javax.persistence.Entity;
import javax.persistence.Table;

@Entity
@Table(name = "users")
public class User {
    private Long id;
    private String name;
}
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	models := frag.SQLAnalysis.ORMModels
	require.NotEmpty(t, models)
	assert.Equal(t, "jpa", models[0].Framework)
	assert.Equal(t, "User", models[0].Model)
}

func TestSQLExtractor_AlembicMigrationDetection(t *testing.T) {
	root := setupTree(t, map[string]string{
		"alembic/versions/001_initial.py":  "# initial",
		"alembic/versions/002_add_email.py": "# add email",
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	mig := frag.SQLAnalysis.Migrations
	require.NotNil(t, mig)
	assert.Equal(t, "alembic/versions", mig.Dir)
	assert.Equal(t, 2, mig.Count)
	assert.Equal(t, "002_add_email.py", mig.Latest)
}

func TestSQLExtractor_NullableColumns(t *testing.T) {
	root := setupTree(t, map[string]string{
		"schema.sql": `
CREATE TABLE items (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT
);
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)
	require.NotNil(t, frag.SQLAnalysis)

	tables := frag.SQLAnalysis.Tables
	require.Len(t, tables, 1)
	cols := tables[0].Columns
	require.Len(t, cols, 3)

	// id: PK -> NotNull=true, Nullable=false
	assert.True(t, cols[0].NotNull)
	assert.False(t, cols[0].Nullable)

	// name: NOT NULL -> NotNull=true, Nullable=false
	assert.True(t, cols[1].NotNull)
	assert.False(t, cols[1].Nullable)

	// description: no constraint -> Nullable=true, NotNull=false
	assert.False(t, cols[2].NotNull)
	assert.True(t, cols[2].Nullable)
}

func TestSQLExtractor_IndexesWithSchema(t *testing.T) {
	root := setupTree(t, map[string]string{
		"schema.sql": `
CREATE UNIQUE INDEX idx_users_email ON public.users(email);
CREATE INDEX idx_orders_user ON orders(user_id);
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	indexes := frag.SQLAnalysis.Indexes
	require.Len(t, indexes, 2)

	assert.Equal(t, "idx_users_email", indexes[0].Name)
	assert.Equal(t, "users", indexes[0].Table)
	assert.True(t, indexes[0].Unique)
	assert.Equal(t, []string{"email"}, indexes[0].Columns)

	assert.Equal(t, "idx_orders_user", indexes[1].Name)
	assert.Equal(t, "orders", indexes[1].Table)
	assert.False(t, indexes[1].Unique)
}

func TestSQLExtractor_ViewsDetection(t *testing.T) {
	root := setupTree(t, map[string]string{
		"schema.sql": `
CREATE VIEW active_users AS SELECT * FROM users WHERE active = true;
CREATE MATERIALIZED VIEW summary AS SELECT count(*) FROM orders;
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	views := frag.SQLAnalysis.Views
	require.Len(t, views, 2)

	assert.Equal(t, "active_users", views[0].Name)
	assert.False(t, views[0].Materialized)

	assert.Equal(t, "summary", views[1].Name)
	assert.True(t, views[1].Materialized)
}

func TestSQLExtractor_GORMTagDetection(t *testing.T) {
	root := setupTree(t, map[string]string{
		"models/item.go": `package models

type Item struct {
	ID   uint
	Name string ` + "`gorm:\"column:name\"`" + `
}
`,
	})

	ext := extract.NewSQLExtractor()
	frag, err := ext.Extract(context.Background(), root)
	require.NoError(t, err)

	models := frag.SQLAnalysis.ORMModels
	require.NotEmpty(t, models, "expected GORM detection via gorm: tag")
	assert.Equal(t, "gorm", models[0].Framework)
}
