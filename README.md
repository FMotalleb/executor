# ⚙️ Executor

**Executor** is a high-performance command-line utility written in Go, designed to **orchestrate and execute parallel processes** efficiently. It supports batching, offset/limit control, parallel execution, and templated command execution.

> Ideal for automating batch jobs, parallel task runners, or any scenario where you need structured control over concurrent executions.

---

## ✨ Features

- 🔁 Parallel execution of custom shell commands
- 📦 Batch processing with offset and limit control
- 🧠 Templated commands using Go templates
- 🕒 Timeout support per command
- 📂 Per-process logging with optional stderr output
- 🛠️ Fully configurable working directory, shell, and logging behavior

---

## 📌 Example Usage

### 🧪 Basic Run

```bash
executor --limit 10000 --batch-size 1000 --processors 5 --log-stderr
```

This runs 10 batches (1000 items per batch) in parallel using 5 workers.

---

### 📜 Custom Command Template

```bash
executor -l 5000 -c 'curl http://localhost/data?start={{ .offset }}&end={{ sum .offset .batchSize }}'
```

Command templates support Go template syntax and include:

- `.offset` → current offset
- `.batchSize` → batch size
- `.limit` → total limit
- `sum` → built-in function for arithmetic

You can see more builtin functions at Template section.

---

### 💬 Logging to stderr

```bash
executor --log-stderr --verbose
```

The default behavior is to log into files with name `exec-<begin>-<end>.log`
Using --log-stderr flag you can see logs of each processor in stderr

Please note that default logging system is pretty chatty if you have seen a bug

---

## 🔧 Flags

```bash
  --batch-size int            Batch size for processing (default 1000)
  -c, --command string        Command to execute (Go template with vars: offset, batchSize, limit) 
                              (default "echo {{ .offset | sum .batchSize }}={{ .limit }}")
  --stdin string              Stdin passed to process (Go template with vars: offset, batchSize, limit) 
                              (default "")
  -l, --limit int             Total number of items to process
  -o, --offset int            Starting offset
  -p, --processors int        Number of parallel executions (default 10)
  --timeout duration          Timeout per command (default 24h0m0s)
  --shell string              Shell to execute commands with (default "/bin/sh")
  --shell-args strings        Shell arguments (default: [-c])
  -w, --working-directory     Working directory (default: current directory)
  --log-dir string            Log file directory (default: current directory)
  --log-stderr                Stream logs to stderr instead of files
  -v, --verbose               Enables verbose logging
  -h, --help                  Display help
```

---

## 🛠 Installation

### 📦 Using `go install`

```bash
go install github.com/FMotalleb/executor@latest
```

Make sure `$GOPATH/bin` is in your `$PATH`.

### 🧪 Build from source

```bash
git clone https://github.com/FMotalleb/executor.git
cd executor
go build -o executor
./executor --help
```

---

## 📁 Example Log Output

When not using `--log-stderr`, logs are written per execution in the specified log directory:

```bash
/home/you/executor/logs/
  ├── run_001.log
  ├── run_002.log
  └── ...
```

---

## 🧩 Template Functions

The `--command` flag uses Go's `text/template` engine, with the following built-in helpers:

| Function       | Description |
|----------------|-------------|
| `env "KEY"`    | Gets environment variable `KEY` |
| `b64enc`       | Base64-encodes a string |
| `b64dec`       | Decodes a base64-encoded string |
| `sum a b`      | Returns `a + b` |
| `toUpper`      | Converts string to UPPERCASE |
| `toLower`      | Converts string to lowercase |
| `trim`         | Trims whitespace from both ends |
| `join list sep`| Joins a list with separator |
| `replace a b`  | Replaces all `a` with `b` in a string |
| `hasPrefix`    | Checks if string has given prefix |
| `hasSuffix`    | Checks if string has given suffix |
| `contains`     | Checks if string contains substring |
| `toJSON`       | Encodes input to JSON |
| `fromJSON`     | Decodes JSON string to map |
| `itoa`         | Converts integer to string |
| `atoi`         | Converts string to integer |
| `toInt`        | Same as `atoi`, converts to int |
| `atob`         | Alias for base64 decode |
