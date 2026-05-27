package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"
)

// ============================================================================
// PARSER FUNCTIONS
// ============================================================================

// parseModelsDirectory parses all Go files in the models directory
func parseModelsDirectory(dir string) ([]ModelInfo, error) {
	fset := token.NewFileSet()
	var models []ModelInfo
	imports := make(map[string]string)

	// Walk through directory
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse file
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		// Extract imports
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			var alias string
			if imp.Name != nil {
				alias = imp.Name.Name
			} else {
				// Extract last component of import path
				parts := strings.Split(importPath, "/")
				alias = parts[len(parts)-1]
			}
			imports[importPath] = alias
		}

		// Extract struct definitions
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.GenDecl:
				if x.Tok == token.TYPE {
					for _, spec := range x.Specs {
						if ts, ok := spec.(*ast.TypeSpec); ok {
							if st, ok := ts.Type.(*ast.StructType); ok {
								model := parseStruct(ts.Name.Name, st, x.Doc, imports)
								if model != nil {
									models = append(models, *model)
								}
							}
						}
					}
				}
			}
			return true
		})

		return nil
	})

	return models, err
}

// parseStruct extracts model information from a struct definition
func parseStruct(structName string, st *ast.StructType, doc *ast.CommentGroup, imports map[string]string) *ModelInfo {
	// Skip non-model structs (request/response types, etc.)
	if !shouldProcessStruct(structName, doc) {
		return nil
	}

	model := &ModelInfo{
		Name:        structName,
		TableName:   inferTableName(structName, doc),
		Comment:     extractComment(doc),
		Fields:      []FieldInfo{},
		PrimaryKey:  []string{},
		Indexes:     []IndexInfo{},
		ForeignKeys: []ForeignKeyInfo{},
	}

	// Parse fields
	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			// Embedded field, skip for now
			continue
		}

		for _, name := range field.Names {
			fieldInfo := parseField(name.Name, field, imports)
			if fieldInfo != nil {
				// Check if primary key before appending
				if isPrimaryKey(fieldInfo, model) {
					fieldInfo.IsPrimaryKey = true
					model.PrimaryKey = append(model.PrimaryKey, fieldInfo.ColumnName)
				}
				model.Fields = append(model.Fields, *fieldInfo)
			}
		}
	}

	// Extract primary key from comments if specified
	extractPrimaryKeyFromComment(model, doc)

	// Generate indexes based on field patterns
	generateIndexes(model)

	// Extract foreign keys from comments
	extractForeignKeys(model, doc)

	return model
}

// shouldProcessStruct determines if a struct should be processed as a model
func shouldProcessStruct(name string, doc *ast.CommentGroup) bool {
	// Skip request/response types
	skipPatterns := []string{"Request", "Response", "Summary", "Input", "Params", "Result"}
	for _, pattern := range skipPatterns {
		if strings.Contains(name, pattern) {
			return false
		}
	}

	// Must have "Table:" comment or be a known model type
	comment := extractComment(doc)
	if strings.Contains(comment, "Table:") {
		return true
	}

	// Known model types
	knownModels := []string{"Approval", "SystemSetting", "Wallet", "WalletTransaction", 
		"WalletTypeInfo", "WalletRule", "Account", "Journal", "LedgerEntry", 
		"Order", "OrderItem", "Product", "User"}
	for _, known := range knownModels {
		if name == known {
			return true
		}
	}

	return false
}

// parseField extracts field information from a struct field
func parseField(fieldName string, field *ast.Field, imports map[string]string) *FieldInfo {
	// Skip fields with db:"-" tag
	tag := extractTag(field.Tag, "db")
	if tag == "-" {
		return nil
	}

	fieldInfo := &FieldInfo{
		Name:       fieldName,
		ColumnName: inferColumnName(fieldName, tag),
		GoType:     getTypeString(field.Type, imports),
		IsNullable: isPointerType(field.Type),
		Comment:    extractComment(field.Doc),
		Tag:        tag,
	}

	// Map Go type to SQL type
	fieldInfo.SQLType = mapGoTypeToSQL(fieldInfo.GoType, fieldInfo.IsNullable, fieldInfo.Comment, fieldInfo.ColumnName)

	// Extract default value from comment
	fieldInfo.DefaultValue = extractDefaultValue(fieldInfo.Comment)

	return fieldInfo
}

// getTypeString returns the string representation of a Go type
func getTypeString(expr ast.Expr, imports map[string]string) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		// Handle qualified types like time.Time
		pkg := x.X.(*ast.Ident).Name
		typ := x.Sel.Name
		return fmt.Sprintf("%s.%s", pkg, typ)
	case *ast.StarExpr:
		// Pointer type
		return "*" + getTypeString(x.X, imports)
	case *ast.ArrayType:
		return "[]" + getTypeString(x.Elt, imports)
	default:
		return "unknown"
	}
}

// isPointerType checks if a type is a pointer
func isPointerType(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}

// extractTag extracts a specific tag value from struct tag
func extractTag(tag *ast.BasicLit, key string) string {
	if tag == nil {
		return ""
	}

	tagStr := strings.Trim(tag.Value, "`")
	// Parse struct tag using reflect
	structTag := reflect.StructTag(tagStr)
	return structTag.Get(key)
}

// inferTableName extracts table name from comment or converts struct name
func inferTableName(structName string, doc *ast.CommentGroup) string {
	comment := extractComment(doc)

	// Look for "Table: table_name" pattern
	if strings.Contains(comment, "Table:") {
		parts := strings.Split(comment, "Table:")
		if len(parts) > 1 {
			tableName := strings.TrimSpace(parts[1])
			// Take first word/line
			lines := strings.Split(tableName, "\n")
			if len(lines) > 0 {
				tableName = strings.TrimSpace(lines[0])
				words := strings.Fields(tableName)
				if len(words) > 0 {
					return words[0]
				}
			}
		}
	}

	// Convert struct name to snake_case table name
	return toSnakeCase(structName)
}

// inferColumnName extracts column name from db tag or converts field name
func inferColumnName(fieldName, dbTag string) string {
	if dbTag != "" && dbTag != "-" {
		return dbTag
	}
	return toSnakeCase(fieldName)
}

// extractComment extracts comment text from comment group
func extractComment(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}

	var parts []string
	for _, comment := range doc.List {
		text := comment.Text
		// Remove comment markers
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

// toSnakeCase converts CamelCase to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// isPrimaryKey checks if a field is a primary key
func isPrimaryKey(field *FieldInfo, model *ModelInfo) bool {
	// Field named ID or Id with int type, or column name is "id"
	if field.ColumnName == "id" && strings.Contains(field.GoType, "int") && !field.IsNullable {
		return true
	}
	// Check comment
	if strings.Contains(strings.ToLower(field.Comment), "primary key") {
		return true
	}
	return false
}

// extractPrimaryKeyFromComment extracts primary key info from struct comment
func extractPrimaryKeyFromComment(model *ModelInfo, doc *ast.CommentGroup) {
	comment := extractComment(doc)
	if strings.Contains(comment, "Primary key:") {
		// Extract primary key definition from comment
		// Format: "Primary key: (user_id, wallet_type)"
		parts := strings.Split(comment, "Primary key:")
		if len(parts) > 1 {
			pkDef := strings.TrimSpace(parts[1])
			// Extract column names from parentheses
			if strings.HasPrefix(pkDef, "(") && strings.HasSuffix(pkDef, ")") {
				pkDef = strings.Trim(pkDef, "()")
				columns := strings.Split(pkDef, ",")
				model.PrimaryKey = []string{}
				for _, col := range columns {
					col = strings.TrimSpace(col)
					if col != "" {
						model.PrimaryKey = append(model.PrimaryKey, col)
					}
				}
			}
		}
	}
}

// generateIndexes generates indexes based on field patterns
func generateIndexes(model *ModelInfo) {
	for _, field := range model.Fields {
		// Index foreign keys (fields ending in _id)
		if strings.HasSuffix(field.ColumnName, "_id") && field.ColumnName != "id" {
			model.Indexes = append(model.Indexes, IndexInfo{
				Name:    fmt.Sprintf("idx_%s_%s", model.TableName, field.ColumnName),
				Columns: []string{field.ColumnName},
				Unique:  false,
			})
		}

		// Index timestamp fields
		if field.ColumnName == "created_at" || field.ColumnName == "updated_at" {
			model.Indexes = append(model.Indexes, IndexInfo{
				Name:    fmt.Sprintf("idx_%s_%s", model.TableName, field.ColumnName),
				Columns: []string{field.ColumnName},
				Unique:  false,
			})
		}
	}
}

// extractForeignKeys extracts foreign key constraints from comments
func extractForeignKeys(model *ModelInfo, doc *ast.CommentGroup) {
	// This is a simplified version - in practice, you might want more sophisticated parsing
	// For now, we'll infer foreign keys from field names ending in _id
	for _, field := range model.Fields {
		if strings.HasSuffix(field.ColumnName, "_id") && field.ColumnName != "id" {
			// Infer referenced table from field name
			// e.g., "user_id" -> "users", "order_id" -> "orders"
			refTable := inferReferencedTable(field.ColumnName)
			if refTable != "" {
				model.ForeignKeys = append(model.ForeignKeys, ForeignKeyInfo{
					Column:    field.ColumnName,
					RefTable:  refTable,
					RefColumn: "id",
					OnDelete:  "RESTRICT",
					OnUpdate:  "RESTRICT",
				})
			}
		}
	}
	
	// Also check comments for explicit foreign key definitions
	comment := extractComment(doc)
	if strings.Contains(comment, "FK:") || strings.Contains(comment, "Foreign key:") {
		// Parse explicit foreign key definitions from comments
		// Format: "FK: column -> table.column" or "Foreign key: column references table"
		// This is a placeholder for future enhancement
	}
}

// inferReferencedTable infers table name from foreign key column name
func inferReferencedTable(columnName string) string {
	// Remove _id suffix
	base := strings.TrimSuffix(columnName, "_id")
	
	// Handle special cases
	specialCases := map[string]string{
		"parent":    "accounts",  // parent_id -> accounts
		"journal":   "journals",   // journal_id -> journals
		"account":   "accounts",   // account_id -> accounts
		"order":     "orders",    // order_id -> orders
		"product":   "products",  // product_id -> products
		"user":      "",           // user_id - skip (users table may not exist)
		"entity":    "",           // entity_id - skip (generic reference)
		"reference": "",           // reference_id - skip (generic reference)
	}
	
	if table, ok := specialCases[base]; ok {
		return table
	}
	
	// Convert to plural (simple heuristic)
	if base != "" && !strings.HasSuffix(base, "s") {
		base = base + "s"
	}
	return base
}

// extractDefaultValue extracts default value from comment
func extractDefaultValue(comment string) string {
	// Look for "DEFAULT" or "default" in comment
	// This is a simplified version
	if strings.Contains(strings.ToLower(comment), "default") {
		// Try to extract value - this would need more sophisticated parsing
		// For now, return empty and let SQL generator handle common defaults
	}
	return ""
}
