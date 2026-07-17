// Package main 是一个本地开发工具脚本，用于查询指定数据库下多张表的字段详细信息。
//
// 使用方式：
//
//	MySQL:
//	  go run scripts/schema_info/main.go -db mysql -dsn "root:123456@tcp(127.0.0.1:3306)/map_server?charset=utf8mb4&parseTime=true&loc=Local" -tables "yard,container"
//
//	PostgreSQL:
//	  go run scripts/schema_info/main.go -db postgres -dsn "host=127.0.0.1 port=5432 user=postgres password=Admin@123 dbname=map_server sslmode=disable" -tables "yard,container"
//
//	查所有表（不指定 -tables）：
//	  go run scripts/schema_info/main.go -db mysql -dsn "root:123456@tcp(127.0.0.1:3306)/map_server?..."
//
// 参数说明：
//
//	-db       数据库类型，必填，可选值：mysql、postgres
//	-dsn      数据库连接字符串，必填
//	-database 数据库名，可选。默认从 DSN 中自动提取
//	-tables   要查询的表名，可选，多个表用逗号分隔。不指定则列出所有表
//	-format   输出格式，可选值：table（表格）、json（JSON）。默认：table
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// ColumnInfo 表示一个字段的详细信息。
type ColumnInfo struct {
	TableName    string  `json:"table_name"`
	ColumnName   string  `json:"column_name"`
	Position     int     `json:"position"`
	DataType     string  `json:"data_type"`
	ColumnType   string  `json:"column_type"`
	IsNullable   string  `json:"is_nullable"`
	DefaultValue *string `json:"default_value"`
	ColumnKey    string  `json:"column_key"`
	Extra        string  `json:"extra"`
	Comment      string  `json:"comment"`
	CharMaxLen   *int64  `json:"char_max_len"`
	NumPrecision *int64  `json:"num_precision"`
	NumScale     *int64  `json:"num_scale"`
}

// Config 脚本配置参数。
type Config struct {
	DBType   string // mysql / postgres
	DSN      string // 连接字符串
	Database string // 数据库名
	Tables   string // 逗号分隔的表名
	Format   string // table / json
}

func main() {
	cfg := parseFlags()

	db, err := sql.Open(cfg.DBType, cfg.DSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "数据库 Ping 失败: %v\n", err)
		os.Exit(1)
	}

	// 如果未指定数据库名，从 DSN 中自动提取
	if cfg.Database == "" {
		cfg.Database = extractDBName(cfg.DBType, cfg.DSN)
	}
	if cfg.Database == "" {
		fmt.Fprintln(os.Stderr, "无法从 DSN 中提取数据库名，请使用 -database 参数指定")
		os.Exit(1)
	}

	fmt.Printf("数据库类型: %s\n", cfg.DBType)
	fmt.Printf("数据库名:   %s\n", cfg.Database)

	// 查询表列信息
	columns, err := queryColumns(db, cfg.DBType, cfg.Database, parseTables(cfg.Tables))
	if err != nil {
		fmt.Fprintf(os.Stderr, "查询失败: %v\n", err)
		os.Exit(1)
	}

	if len(columns) == 0 {
		fmt.Println("未找到任何表或字段信息，请检查数据库名和表名是否正确。")
		return
	}

	// 输出
	switch cfg.Format {
	case "json":
		outputJSON(columns)
	default:
		outputTable(columns)
	}
}

func parseFlags() Config {
	dbType := flag.String("db", "", "数据库类型 (mysql / postgres)")
	dsn := flag.String("dsn", "", "数据库连接字符串")
	database := flag.String("database", "", "数据库名（可选，默认从 DSN 中自动提取）")
	tables := flag.String("tables", "", "要查询的表名，多个表用逗号分隔（可选，不指定则查询所有表）")
	format := flag.String("format", "table", "输出格式 (table / json)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: go run scripts/schema_info/main.go [参数]\n\n")
		fmt.Fprintf(os.Stderr, "参数:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  MySQL 指定表:\n")
		fmt.Fprintf(os.Stderr, "    go run scripts/schema_info/main.go -db mysql -dsn \"root:123456@tcp(127.0.0.1:3306)/map_server?...\" -tables \"yard,container\"\n")
		fmt.Fprintf(os.Stderr, "  PostgreSQL 所有表:\n")
		fmt.Fprintf(os.Stderr, "    go run scripts/schema_info/main.go -db postgres -dsn \"host=127.0.0.1 port=5432 user=postgres password=123 dbname=map_server sslmode=disable\"\n")
		fmt.Fprintf(os.Stderr, "  JSON 格式输出:\n")
		fmt.Fprintf(os.Stderr, "    go run scripts/schema_info/main.go -db mysql -dsn \"...\" -format json\n")
	}

	flag.Parse()

	if *dbType == "" {
		fmt.Fprintln(os.Stderr, "错误: -db 参数必填 (mysql / postgres)")
		flag.Usage()
		os.Exit(1)
	}
	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "错误: -dsn 参数必填")
		flag.Usage()
		os.Exit(1)
	}
	*dbType = strings.ToLower(*dbType)
	if *dbType != "mysql" && *dbType != "postgres" {
		fmt.Fprintf(os.Stderr, "错误: -db 只支持 mysql 或 postgres，当前值: %s\n", *dbType)
		os.Exit(1)
	}
	if *format != "table" && *format != "json" {
		fmt.Fprintf(os.Stderr, "错误: -format 只支持 table 或 json，当前值: %s\n", *format)
		os.Exit(1)
	}

	return Config{
		DBType:   *dbType,
		DSN:      *dsn,
		Database: *database,
		Tables:   *tables,
		Format:   *format,
	}
}

// extractDBName 从 DSN 中提取数据库名。
func extractDBName(dbType, dsn string) string {
	switch dbType {
	case "mysql":
		// root:password@tcp(host:port)/dbname?params
		re := regexp.MustCompile(`/[^/?]*`)
		if m := re.FindString(dsn); m != "" {
			return strings.TrimPrefix(m, "/")
		}
	case "postgres":
		// ... dbname=xxx ...
		re := regexp.MustCompile(`dbname=([^\s]+)`)
		if m := re.FindStringSubmatch(dsn); len(m) >= 2 {
			return m[1]
		}
	}
	return ""
}

// parseTables 解析逗号分隔的表名列表。
func parseTables(tablesStr string) []string {
	if tablesStr == "" {
		return nil // nil 表示查询所有表
	}
	parts := strings.Split(tablesStr, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// queryColumns 查询指定数据库和表的字段信息。
func queryColumns(db *sql.DB, dbType, database string, tables []string) ([]ColumnInfo, error) {
	switch dbType {
	case "mysql":
		return queryMySQLColumns(db, database, tables)
	case "postgres":
		return queryPostgresColumns(db, database, tables)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}

// queryMySQLColumns 查询 MySQL 数据库的字段信息。
func queryMySQLColumns(db *sql.DB, database string, tables []string) ([]ColumnInfo, error) {
	query := `
		SELECT
			TABLE_NAME,
			COLUMN_NAME,
			ORDINAL_POSITION,
			DATA_TYPE,
			COLUMN_TYPE,
			IS_NULLABLE,
			COLUMN_DEFAULT,
			COLUMN_KEY,
			EXTRA,
			COLUMN_COMMENT,
			CHARACTER_MAXIMUM_LENGTH,
			NUMERIC_PRECISION,
			NUMERIC_SCALE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, ORDINAL_POSITION
	`

	if len(tables) > 0 {
		placeholders := make([]string, len(tables))
		for i := range tables {
			placeholders[i] = "?"
		}
		query = `
			SELECT
				TABLE_NAME,
				COLUMN_NAME,
				ORDINAL_POSITION,
				DATA_TYPE,
				COLUMN_TYPE,
				IS_NULLABLE,
				COLUMN_DEFAULT,
				COLUMN_KEY,
				EXTRA,
				COLUMN_COMMENT,
				CHARACTER_MAXIMUM_LENGTH,
				NUMERIC_PRECISION,
				NUMERIC_SCALE
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = ? AND TABLE_NAME IN (` + strings.Join(placeholders, ",") + `)
			ORDER BY TABLE_NAME, ORDINAL_POSITION
		`
	}

	args := make([]any, 0, len(tables)+1)
	args = append(args, database)
	for _, t := range tables {
		args = append(args, t)
	}

	return executeQuery(db, query, args...)
}

// queryPostgresColumns 查询 PostgreSQL 数据库的字段信息。
func queryPostgresColumns(db *sql.DB, database string, tables []string) ([]ColumnInfo, error) {
	// PostgreSQL 中 information_schema 的列名是小写的
	query := `
		SELECT
			c.table_name,
			c.column_name,
			c.ordinal_position,
			c.data_type,
			c.udt_name,
			c.is_nullable,
			c.column_default,
			'' AS column_key,
			'' AS extra,
			COALESCE(pd.description, '') AS column_comment,
			c.character_maximum_length,
			c.numeric_precision,
			c.numeric_scale
		FROM information_schema.columns c
		LEFT JOIN pg_catalog.pg_statio_all_tables st
			ON c.table_schema = st.schemaname AND c.table_name = st.relname
		LEFT JOIN pg_catalog.pg_description pd
			ON pd.objoid = st.relid AND pd.objsubid = c.ordinal_position
		WHERE c.table_catalog = $1
			AND c.table_schema NOT IN ('pg_catalog', 'information_schema')
	`

	args := []any{database}

	if len(tables) > 0 {
		placeholders := make([]string, len(tables))
		for i := range tables {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
		}
		query += ` AND c.table_name IN (` + strings.Join(placeholders, ",") + `)`
		for _, t := range tables {
			args = append(args, t)
		}
	}

	query += ` ORDER BY c.table_name, c.ordinal_position`

	return executeQuery(db, query, args...)
}

// executeQuery 执行查询并将结果映射为 ColumnInfo 切片。
func executeQuery(db *sql.DB, query string, args ...any) ([]ColumnInfo, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var c ColumnInfo
		if err := rows.Scan(
			&c.TableName,
			&c.ColumnName,
			&c.Position,
			&c.DataType,
			&c.ColumnType,
			&c.IsNullable,
			&c.DefaultValue,
			&c.ColumnKey,
			&c.Extra,
			&c.Comment,
			&c.CharMaxLen,
			&c.NumPrecision,
			&c.NumScale,
		); err != nil {
			return nil, fmt.Errorf("扫描行数据失败: %w", err)
		}
		columns = append(columns, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果集失败: %w", err)
	}
	return columns, nil
}

// outputTable 以表格形式输出字段信息。
func outputTable(columns []ColumnInfo) {
	// 按表名分组
	tables := make(map[string][]ColumnInfo)
	var order []string
	for _, c := range columns {
		if _, ok := tables[c.TableName]; !ok {
			order = append(order, c.TableName)
		}
		tables[c.TableName] = append(tables[c.TableName], c)
	}

	// P M P P M P P M P M  = 对齐中英文混合宽度
	for _, tbl := range order {
		cols := tables[tbl]
		fmt.Printf("\n━━━ 表: %s (%d 个字段) ━━━\n", tbl, len(cols))
		// 表头
		fmt.Printf("%-4s %-24s %-20s %-8s %-8s %-22s %-20s\n",
			"序号", "字段名", "列类型", "可空", "键", "默认值", "注释")
		fmt.Println(strings.Repeat("─", 115))
		for _, c := range cols {
			nullable := c.IsNullable
			if nullable == "YES" {
				nullable = "是"
			} else {
				nullable = "否"
			}
			key := c.ColumnKey
			if key == "" {
				key = "-"
			} else if key == "PRI" {
				key = "主键"
			} else if key == "UNI" {
				key = "唯一"
			} else if key == "MUL" {
				key = "索引"
			}

			defaultVal := "-"
			if c.DefaultValue != nil && *c.DefaultValue != "" {
				defaultVal = truncateStr(*c.DefaultValue, 20)
			}

			comment := truncateStr(c.Comment, 20)

			fmt.Printf("%-4d %-24s %-20s %-8s %-8s %-22s %-20s\n",
				c.Position, c.ColumnName, c.ColumnType, nullable, key, defaultVal, comment)
		}
		fmt.Printf("表 %s 共 %d 个字段\n\n", tbl, len(cols))
	}
}

// outputJSON 以 JSON 格式输出字段信息。
func outputJSON(columns []ColumnInfo) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(columns); err != nil {
		fmt.Fprintf(os.Stderr, "JSON 编码失败: %v\n", err)
		os.Exit(1)
	}
}

// truncateStr 截断字符串，超长尾部加 "…"。
func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "…"
	}
	return s
}
