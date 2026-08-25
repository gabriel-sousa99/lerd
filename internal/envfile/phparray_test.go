package envfile

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const magentoEnvPHP = `<?php
return [
    'backend' => [
        'frontName' => 'admin'
    ],
    'db' => [
        'connection' => [
            'default' => [
                'host' => 'localhost',
                'dbname' => 'magento',
                'username' => 'root',
                'password' => 'secret',
                'active' => '1'
            ]
        ],
        'table_prefix' => ''
    ],
    'x-frame-options' => 'SAMEORIGIN',
    'MAGE_MODE' => 'default',
    'cache_types' => [
        'config' => 1,
        'layout' => 0
    ],
    'downloadable_domains' => [
        'magento.test'
    ],
    'install' => [
        'date' => 'Thu, 09 Jul 2026 10:00:00 +0000'
    ]
];
`

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "env.php")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadPhpArrayFlattensNestedKeys(t *testing.T) {
	got, err := ReadPhpArray(writeTemp(t, magentoEnvPHP))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"backend.frontName":              "admin",
		"db.connection.default.host":     "localhost",
		"db.connection.default.dbname":   "magento",
		"db.connection.default.username": "root",
		"db.connection.default.password": "secret",
		"db.connection.default.active":   "1",
		"db.table_prefix":                "",
		"x-frame-options":                "SAMEORIGIN",
		"MAGE_MODE":                      "default",
		"cache_types.config":             "1",
		"cache_types.layout":             "0",
		"downloadable_domains.0":         "magento.test",
		"install.date":                   "Thu, 09 Jul 2026 10:00:00 +0000",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d keys, want %d: %v", len(got), len(want), got)
	}
}

func TestReadPhpArrayHandlesSyntaxVariants(t *testing.T) {
	src := `<?php
// a comment with ] and '
# hash comment
/* block ' comment */
return array(
    "double" => "quoted",
    'esc' => 'it\'s',
    'num' => 3306,
    'yes' => true,
    'no' => false,
    'nil' => null,
    'trailing' => array('a' => 1,),
);
`
	got, err := ReadPhpArray(writeTemp(t, src))
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{
		"double": "quoted", "esc": "it's", "num": "3306",
		"yes": "true", "no": "false", "nil": "", "trailing.a": "1",
	} {
		if got[k] != want {
			t.Errorf("%s = %q, want %q", k, got[k], want)
		}
	}
}

// A `return` sitting inside a string literal before the real return statement
// must not be mistaken for it, or the scanner parses the wrong value.
func TestReadPhpArraySkipsReturnInsideString(t *testing.T) {
	src := `<?php
$note = 'remember to return the array';
return [
    'db' => ['host' => 'localhost'],
];
`
	got, err := ReadPhpArray(writeTemp(t, src))
	if err != nil {
		t.Fatal(err)
	}
	if got["db.host"] != "localhost" {
		t.Fatalf("scanner matched a return inside a string: %v", got)
	}
}

func TestReadPhpArrayMissingOrEmptyFile(t *testing.T) {
	if _, err := ReadPhpArray(filepath.Join(t.TempDir(), "nope.php")); err == nil {
		t.Error("missing file should error")
	}
	got, err := ReadPhpArray(writeTemp(t, "<?php\n"))
	if err != nil {
		t.Fatalf("file with no return should not error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestApplyPhpArrayUpdatesRewritesNestedValue(t *testing.T) {
	p := writeTemp(t, magentoEnvPHP)
	err := ApplyPhpArrayUpdates(p, map[string]string{
		"db.connection.default.host":     "lerd-mysql",
		"db.connection.default.dbname":   "magento_test",
		"db.connection.default.password": "lerd",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadPhpArray(p)
	if err != nil {
		t.Fatal(err)
	}
	if got["db.connection.default.host"] != "lerd-mysql" {
		t.Errorf("host = %q", got["db.connection.default.host"])
	}
	if got["db.connection.default.dbname"] != "magento_test" {
		t.Errorf("dbname = %q", got["db.connection.default.dbname"])
	}
	// Untouched keys survive.
	if got["backend.frontName"] != "admin" || got["install.date"] == "" {
		t.Errorf("unrelated keys lost: %v", got)
	}

	// The php-const writer appends a dead define() after the return. This one
	// must never do that.
	body, _ := os.ReadFile(p)
	if strings.Contains(string(body), "define(") {
		t.Errorf("appended dead code:\n%s", body)
	}
	if strings.Count(string(body), "return") != 1 {
		t.Errorf("expected exactly one return:\n%s", body)
	}
}

func TestApplyPhpArrayUpdatesCreatesMissingPath(t *testing.T) {
	p := writeTemp(t, magentoEnvPHP)
	err := ApplyPhpArrayUpdates(p, map[string]string{
		"system.default.catalog.search.engine":                     "opensearch",
		"system.default.catalog.search.opensearch_server_hostname": "lerd-opensearch",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := ReadPhpArray(p)
	if got["system.default.catalog.search.engine"] != "opensearch" {
		t.Fatalf("engine = %q\n", got["system.default.catalog.search.engine"])
	}
	if got["system.default.catalog.search.opensearch_server_hostname"] != "lerd-opensearch" {
		t.Fatalf("hostname = %q", got["system.default.catalog.search.opensearch_server_hostname"])
	}
	if got["db.connection.default.host"] != "localhost" {
		t.Error("existing keys clobbered")
	}
}

// Writing then reading then writing must converge, or `lerd env` would churn
// the file on every run.
func TestApplyPhpArrayUpdatesIsIdempotent(t *testing.T) {
	p := writeTemp(t, magentoEnvPHP)
	up := map[string]string{"db.connection.default.host": "lerd-mysql"}
	if err := ApplyPhpArrayUpdates(p, up); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(p)
	if err := ApplyPhpArrayUpdates(p, up); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(p)
	if string(first) != string(second) {
		t.Errorf("not idempotent:\n--- first\n%s\n--- second\n%s", first, second)
	}
}

// Types are preserved: an int stays an int, a bool stays a bool, so Magento's
// own config reader sees what it expects.
func TestApplyPhpArrayUpdatesPreservesScalarTypes(t *testing.T) {
	p := writeTemp(t, "<?php\nreturn [\n    'port' => 3306,\n    'on' => true,\n    'name' => 'x',\n];\n")
	if err := ApplyPhpArrayUpdates(p, map[string]string{"port": "3307", "on": "false", "name": "y"}); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(p)
	s := string(body)
	if !strings.Contains(s, "'port' => 3307,") {
		t.Errorf("port should stay an int:\n%s", s)
	}
	if !strings.Contains(s, "'on' => false,") {
		t.Errorf("bool should stay a bool:\n%s", s)
	}
	if !strings.Contains(s, "'name' => 'y',") {
		t.Errorf("string should stay quoted:\n%s", s)
	}
}

func TestApplyPhpArrayUpdatesCreatesFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "app", "etc", "env.php")
	if err := ApplyPhpArrayUpdates(p, map[string]string{"db.connection.default.host": "lerd-mysql"}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPhpArray(p)
	if err != nil {
		t.Fatal(err)
	}
	if got["db.connection.default.host"] != "lerd-mysql" {
		t.Fatalf("got %v", got)
	}
	body, _ := os.ReadFile(p)
	if !strings.HasPrefix(string(body), "<?php") {
		t.Errorf("missing php open tag:\n%s", body)
	}
}

// A value containing a quote or backslash must round-trip, not break the file.
func TestApplyPhpArrayUpdatesEscapesValues(t *testing.T) {
	p := writeTemp(t, "<?php\nreturn ['a' => 'x'];\n")
	if err := ApplyPhpArrayUpdates(p, map[string]string{"a": `it's a \ back`}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPhpArray(p)
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != `it's a \ back` {
		t.Fatalf("got %q", got["a"])
	}
}

func TestPhpArrayHandlesFloats(t *testing.T) {
	p := writeTemp(t, "<?php\nreturn ['ratio' => 1.5, 'neg' => -2.25, 'sci' => 1.0e-5, 'i' => 7];\n")
	got, err := ReadPhpArray(p)
	if err != nil {
		t.Fatal(err)
	}
	if got["ratio"] != "1.5" || got["neg"] != "-2.25" || got["i"] != "7" {
		t.Fatalf("got %v", got)
	}
	if err := ApplyPhpArrayUpdates(p, map[string]string{"ratio": "2.75"}); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(p)
	if !strings.Contains(string(body), "'ratio' => 2.75,") {
		t.Fatalf("float not preserved:\n%s", body)
	}
}

func TestReaderDispatchesOnFormat(t *testing.T) {
	arr := writeTemp(t, "<?php\nreturn ['db' => ['host' => 'lerd-mysql']];\n")
	if got := Reader(arr, "php-array")("db.host"); got != "lerd-mysql" {
		t.Errorf("php-array reader: %q", got)
	}
	konst := writeTemp(t, "<?php\ndefine('DB_HOST', 'lerd-mysql');\n")
	if got := Reader(konst, "php-const")("DB_HOST"); got != "lerd-mysql" {
		t.Errorf("php-const reader: %q", got)
	}
	dot := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(dot, []byte("DB_HOST=lerd-mysql\n"), 0o644)
	if got := Reader(dot, "dotenv")("DB_HOST"); got != "lerd-mysql" {
		t.Errorf("dotenv reader: %q", got)
	}
	// A missing file must not panic or error out the caller.
	if got := Reader(filepath.Join(t.TempDir(), "nope.php"), "php-array")("x"); got != "" {
		t.Errorf("missing file: %q", got)
	}
}

func TestApplyPhpArrayUpdates_NoOpWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env.php")

	if err := ApplyPhpArrayUpdates(path, map[string]string{"db.host": "lerd-mysql"}); err != nil {
		t.Fatalf("first write: %v", err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(path)

	// Re-applying the values the file already holds must not touch it: the writer
	// reprints the whole file, so a rewrite would churn a Magento deployment config
	// on every worktree sync.
	time.Sleep(10 * time.Millisecond)
	if err := ApplyPhpArrayUpdates(path, map[string]string{"db.host": "lerd-mysql"}); err != nil {
		t.Fatalf("second write: %v", err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Errorf("unchanged content rewrote the file: mtime %v -> %v", before.ModTime(), after.ModTime())
	}
	if now, _ := os.ReadFile(path); string(now) != string(body) {
		t.Errorf("content changed on a no-op write")
	}

	// A real change still lands.
	if err := ApplyPhpArrayUpdates(path, map[string]string{"db.host": "lerd-mariadb-11-8"}); err != nil {
		t.Fatalf("third write: %v", err)
	}
	got, _ := ReadPhpArray(path)
	if got["db.host"] != "lerd-mariadb-11-8" {
		t.Errorf("db.host = %q, want lerd-mariadb-11-8", got["db.host"])
	}
}

// A returning-array config is read by people as much as by code: CakePHP's
// app_local.php is mostly guidance on what each key is for, and its values call
// a function it imports at the top. Rewriting a value must leave all of that
// where it was, so the writer edits spans rather than reprinting the tree.
const cakeAppLocalPHP = `<?php

use function Cake\Core\env;

/*
 * Local configuration file.
 */
return [
    'debug' => filter_var(env('DEBUG', true), FILTER_VALIDATE_BOOLEAN),

    'Datasources' => [
        'default' => [
            'host' => 'localhost',
            /*
             * MAMP users will want to set the port.
             */
            //'port' => 'non_standard_port_number',

            'username' => 'my_app',
            'database' => 'my_app',

            'url' => env('DATABASE_URL', null),
        ],
    ],
];
`

func TestApplyPhpArrayUpdates_KeepsCommentsAndExpressions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app_local.php")
	if err := os.WriteFile(path, []byte(cakeAppLocalPHP), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyPhpArrayUpdates(path, map[string]string{
		"Datasources.default.host":     "lerd-mysql",
		"Datasources.default.database": "fourth",
	}); err != nil {
		t.Fatalf("ApplyPhpArrayUpdates: %v", err)
	}
	body, _ := os.ReadFile(path)
	got := string(body)

	for _, keep := range []string{
		"use function Cake\\Core\\env;",
		"* Local configuration file.",
		"* MAMP users will want to set the port.",
		"//'port' => 'non_standard_port_number',",
		"filter_var(env('DEBUG', true), FILTER_VALIDATE_BOOLEAN)",
		"env('DATABASE_URL', null)",
	} {
		if !strings.Contains(got, keep) {
			t.Errorf("rewrite dropped %q:\n%s", keep, got)
		}
	}
	if !strings.Contains(got, "'host' => 'lerd-mysql',") {
		t.Errorf("host was not rewritten:\n%s", got)
	}
	if strings.Contains(got, "'my_app'") && !strings.Contains(got, "'username' => 'my_app'") {
		t.Errorf("rewrite disturbed a value it was not given:\n%s", got)
	}
}

// A key the file does not have yet is inserted into the array it belongs to,
// indented with its siblings, rather than reprinting the array around it.
func TestApplyPhpArrayUpdates_InsertsMissingKeyInPlace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app_local.php")
	if err := os.WriteFile(path, []byte(cakeAppLocalPHP), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyPhpArrayUpdates(path, map[string]string{"Datasources.default.port": "3306"}); err != nil {
		t.Fatalf("ApplyPhpArrayUpdates: %v", err)
	}
	body, _ := os.ReadFile(path)
	if !strings.Contains(string(body), "\n            'port' => '3306',\n") {
		t.Errorf("port was not inserted alongside its siblings:\n%s", body)
	}
	if !strings.Contains(string(body), "use function Cake\\Core\\env;") {
		t.Errorf("insertion cost the file its import:\n%s", body)
	}
	vals, err := ReadPhpArray(path)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if vals["Datasources.default.port"] != "3306" {
		t.Errorf("port = %q, want 3306", vals["Datasources.default.port"])
	}
}

// Several new keys under one absent parent produce a single entry for it, not
// one per key, which would leave the array holding the same name twice.
func TestApplyPhpArrayUpdates_GroupsKeysUnderOneNewParent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app_local.php")
	if err := os.WriteFile(path, []byte(cakeAppLocalPHP), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyPhpArrayUpdates(path, map[string]string{
		"Datasources.replica.host":     "lerd-mysql",
		"Datasources.replica.database": "fourth",
	}); err != nil {
		t.Fatalf("ApplyPhpArrayUpdates: %v", err)
	}
	body, _ := os.ReadFile(path)
	if n := strings.Count(string(body), "'replica' =>"); n != 1 {
		t.Errorf("replica written %d times, want once:\n%s", n, body)
	}
	vals, _ := ReadPhpArray(path)
	if vals["Datasources.replica.host"] != "lerd-mysql" || vals["Datasources.replica.database"] != "fourth" {
		t.Errorf("replica values did not survive: %v", vals)
	}
}

// Several keys descending through the same value that is not an array, as an
// app_local.php reading its datasource from env() has. One replacement serves
// them all, or the second splices over the first and the file stops parsing.
func TestApplyPhpArrayUpdates_KeysThroughOneNonArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app_local.php")
	body := "<?php\nuse function Cake\\Core\\env;\nreturn [\n    'Datasources' => env('DATABASE_URL'),\n    'debug' => true,\n];\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyPhpArrayUpdates(path, map[string]string{
		"Datasources.default.database": "site",
		"Datasources.default.host":     "lerd-mysql",
		"Datasources.default.username": "lerd",
	}); err != nil {
		t.Fatalf("ApplyPhpArrayUpdates: %v", err)
	}
	vals, err := ReadPhpArray(path)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	for key, want := range map[string]string{
		"Datasources.default.database": "site",
		"Datasources.default.host":     "lerd-mysql",
		"Datasources.default.username": "lerd",
	} {
		if vals[key] != want {
			out, _ := os.ReadFile(path)
			t.Errorf("%s = %q, want %q:\n%s", key, vals[key], want, out)
		}
	}
	if out, _ := os.ReadFile(path); !strings.Contains(string(out), "'debug' => true") {
		t.Errorf("the rest of the file did not survive:\n%s", out)
	}
}

// A value the reader cannot evaluate is nobody's to report: the key is absent
// from a read rather than carrying the source text as if it were the value.
func TestReadPhpArray_OmitsExpressionValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app_local.php")
	if err := os.WriteFile(path, []byte(cakeAppLocalPHP), 0644); err != nil {
		t.Fatal(err)
	}

	vals, err := ReadPhpArray(path)
	if err != nil {
		t.Fatalf("ReadPhpArray: %v", err)
	}
	if _, ok := vals["debug"]; ok {
		t.Errorf("debug = %q, want no value for an expression", vals["debug"])
	}
	if _, ok := vals["Datasources.default.url"]; ok {
		t.Errorf("url = %q, want no value for an expression", vals["Datasources.default.url"])
	}
	if vals["Datasources.default.host"] != "localhost" {
		t.Errorf("host = %q, want the literal alongside the expressions", vals["Datasources.default.host"])
	}
}

// phpLint runs php -l over the file where php is on PATH, as an extra layer on
// top of the always-on assertions: a splice landing mid-expression is a file
// only php calls wrong. Absence of php weakens nothing below, it only skips
// this one extra check.
func phpLint(t *testing.T, path string) {
	t.Helper()
	bin, err := exec.LookPath("php")
	if err != nil {
		return
	}
	out, err := exec.Command(bin, "-l", path).CombinedOutput()
	if err != nil {
		body, _ := os.ReadFile(path)
		t.Fatalf("php rejected the rewritten file: %v\n%s\n--- file ---\n%s", err, out, body)
	}
}

// A key naming a node and another key descending through it cannot both hold:
// the node is either the scalar or the array. The deeper key is the one that
// makes the other's parent exist, so it wins, and the file must stay parseable
// with the rest of it intact.
func TestApplyPhpArrayUpdates_KeyNamingANodeAnotherDescendsThrough(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app_local.php")
	body := "<?php\nreturn [\n    'Datasources' => null,\n    'debug' => true,\n];\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyPhpArrayUpdates(path, map[string]string{
		"Datasources":                  "x",
		"Datasources.default.database": "site",
	}); err != nil {
		t.Fatalf("ApplyPhpArrayUpdates: %v", err)
	}
	phpLint(t, path)

	vals, err := ReadPhpArray(path)
	if err != nil {
		out, _ := os.ReadFile(path)
		t.Fatalf("re-read: %v\n%s", err, out)
	}
	if vals["Datasources.default.database"] != "site" {
		t.Errorf("the deeper key did not survive: %v", vals)
	}
	if vals["Datasources"] == "x" {
		t.Errorf("the shallower key overwrote the array the deeper one needs: %v", vals)
	}
	if vals["debug"] != "true" {
		t.Errorf("an unrelated key was damaged: %v", vals)
	}
}

// The same collision through an array node: the key naming the whole array is
// dropped in favour of the insertion into it, and the insertion lands.
func TestApplyPhpArrayUpdates_WholeArrayAndAnInsertionIntoIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app_local.php")
	if err := os.WriteFile(path, []byte(cakeAppLocalPHP), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyPhpArrayUpdates(path, map[string]string{
		"Datasources.default":      "x",
		"Datasources.default.port": "3306",
	}); err != nil {
		t.Fatalf("ApplyPhpArrayUpdates: %v", err)
	}
	phpLint(t, path)

	vals, err := ReadPhpArray(path)
	if err != nil {
		out, _ := os.ReadFile(path)
		t.Fatalf("re-read: %v\n%s", err, out)
	}
	if vals["Datasources.default.port"] != "3306" {
		t.Errorf("the insertion was lost: %v", vals)
	}
	if vals["Datasources.default"] == "x" {
		t.Errorf("the array was overwritten by the shallower key: %v", vals)
	}
	out, _ := os.ReadFile(path)
	if !strings.Contains(string(out), "use function Cake\\Core\\env;") {
		t.Errorf("the file lost its import:\n%s", out)
	}
}

// A scalar update inside a single-line array and a new key grafted into the
// same array are two edits into one span of the file. Neither may be lost.
func TestApplyPhpArrayUpdates_EditAndGraftInOneSingleLineArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app_local.php")
	body := "<?php\nreturn [\n    'Datasources' => ['host' => 'localhost'],\n    'debug' => true,\n];\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyPhpArrayUpdates(path, map[string]string{
		"Datasources.host": "db",
		"Datasources.port": "3306",
	}); err != nil {
		t.Fatalf("ApplyPhpArrayUpdates: %v", err)
	}
	phpLint(t, path)

	vals, err := ReadPhpArray(path)
	if err != nil {
		out, _ := os.ReadFile(path)
		t.Fatalf("re-read: %v\n%s", err, out)
	}
	if vals["Datasources.host"] != "db" || vals["Datasources.port"] != "3306" {
		out, _ := os.ReadFile(path)
		t.Errorf("an edit was lost: %v\n%s", vals, out)
	}
}

// A last entry written without a trailing comma is how plenty of hand-edited
// configs read. Grafting after it must supply the comma PHP needs between them.
func TestApplyPhpArrayUpdates_GraftAfterEntryWithoutTrailingComma(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env.php")
	body := "<?php\nreturn [\n    'db' => [\n        'host' => 'localhost'\n    ]\n];\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyPhpArrayUpdates(path, map[string]string{"db.port": "3306"}); err != nil {
		t.Fatalf("ApplyPhpArrayUpdates: %v", err)
	}
	phpLint(t, path)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "<?php\nreturn [\n    'db' => [\n        'host' => 'localhost',\n        'port' => '3306',\n    ]\n];\n"
	if string(got) != want {
		t.Errorf("rewritten file:\n%s\nwant:\n%s", got, want)
	}
}

// A negative constant is an expression, not the number '-'. It reads as no
// value and reprints untouched, like every other expression.
func TestReadPhpArray_NegativeConstantIsAnExpression(t *testing.T) {
	path := writeTemp(t, "<?php\nreturn [\n    'a' => ['limit' => -PHP_INT_MAX],\n];\n")
	vals, err := ReadPhpArray(path)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := vals["a.limit"]; ok {
		t.Errorf("an expression reported a value: %q", v)
	}
	if _, ok := vals["a.0"]; ok {
		t.Errorf("a phantom positional entry appeared: %v", vals)
	}

	if err := ApplyPhpArrayUpdates(path, map[string]string{"a.extra": "1"}); err != nil {
		t.Fatalf("ApplyPhpArrayUpdates: %v", err)
	}
	phpLint(t, path)
	out, _ := os.ReadFile(path)
	if !strings.Contains(string(out), "-PHP_INT_MAX") {
		t.Errorf("the expression did not survive the rewrite:\n%s", out)
	}
	vals, err = ReadPhpArray(path)
	if err != nil {
		t.Fatalf("re-read: %v\n%s", err, out)
	}
	if vals["a.extra"] != "1" {
		t.Errorf("the added key was lost: %v", vals)
	}
}

// A file whose entries are not comma-separated is not PHP, however it got that
// way. Reading it as if it were hides exactly the corruption the writer's own
// guard exists to catch.
func TestReadPhpArray_RefusesMissingCommaBetweenEntries(t *testing.T) {
	path := writeTemp(t, "<?php\nreturn [\n    'a' => 1\n    'b' => 2,\n];\n")
	if _, err := ReadPhpArray(path); err == nil {
		t.Error("a file php rejects was read as if it parsed")
	}
}
