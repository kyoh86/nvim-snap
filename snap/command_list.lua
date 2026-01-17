local case_def = require("snap.case_def")
local util = require("snap.util")

local M = {}

local function format_path(path)
  local cwd = vim.fn.getcwd()
  local rel = vim.fn.fnamemodify(path, ":.")
  if rel == "" or rel:sub(1, 2) == ".." then
    return vim.fs.normalize(path)
  end
  if cwd == path then
    return "."
  end
  return rel
end

local function usage()
  return table.concat({
    "usage:",
    "  nvim-snap list [options]",
    "",
    "options:",
    "  --root PATH       Root directory to search (default: .)",
    "  --cases-dir PATH  Cases directory under root (default: snapcase)",
    "  --tag TAG      Filter by tag (repeatable, comma-separated)",
    "  --case NAME    Filter by case name (repeatable, comma-separated)",
    "  --json         Output JSON",
    "  -h, --help     Show this help",
  }, "\n")
end

local function split_values(value)
  local out = {}
  for item in string.gmatch(value, "([^,]+)") do
    local trimmed = vim.trim(item)
    if trimmed ~= "" then
      table.insert(out, trimmed)
    end
  end
  return out
end

local function parse_args(args)
  local opts = {
    root = ".",
    cases_dir = "snapcase",
    tags = {},
    cases = {},
    json = false,
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--root" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--root requires a value")
      else
        opts.root = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--root=") then
      opts.root = string.sub(arg, 8)
    elseif arg == "--cases-dir" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--cases-dir requires a value")
      else
        opts.cases_dir = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--cases-dir=") then
      opts.cases_dir = string.sub(arg, 13)
    elseif arg == "--tag" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--tag requires a value")
      else
        vim.list_extend(opts.tags, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--tag=") then
      vim.list_extend(opts.tags, split_values(string.sub(arg, 7)))
    elseif arg == "--case" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--case requires a value")
      else
        vim.list_extend(opts.cases, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--case=") then
      vim.list_extend(opts.cases, split_values(string.sub(arg, 8)))
    elseif arg == "--json" then
      opts.json = true
    elseif arg == "--help" or arg == "-h" then
      opts.help = true
    else
      opts.unknown = opts.unknown or {}
      table.insert(opts.unknown, arg)
    end
    i = i + 1
  end
  return opts
end

local function print_text(cases)
  local rows = {
    { "name", "title", "kind", "tags", "path" },
  }
  for _, c in ipairs(cases) do
    table.insert(rows, {
      c.name,
      c.title,
      c.kind,
      table.concat(c.tags, ","),
      format_path(c.path),
    })
  end
  local widths = { 0, 0, 0, 0, 0 }
  for _, row in ipairs(rows) do
    for idx = 1, #widths do
      local width = vim.fn.strdisplaywidth(row[idx] or "")
      if width > widths[idx] then
        widths[idx] = width
      end
    end
  end
  local function pad(value, width)
    local text = value or ""
    local pad_len = width - vim.fn.strdisplaywidth(text)
    if pad_len < 0 then
      pad_len = 0
    end
    return text .. string.rep(" ", pad_len)
  end
  for _, row in ipairs(rows) do
    print(table.concat({
      pad(row[1], widths[1]),
      pad(row[2], widths[2]),
      pad(row[3], widths[3]),
      pad(row[4], widths[4]),
      pad(row[5], widths[5]),
    }, "  "))
  end
end

local function print_json(root, cases)
  local out = {
    root = root,
    cases = {},
  }
  for _, c in ipairs(cases) do
    table.insert(out.cases, {
      name = c.name,
      title = c.title,
      kind = c.kind,
      tags = c.tags,
      path = vim.fs.normalize(c.path),
    })
  end
  print(vim.json.encode(out, { indent = "  " }))
end

function M.run(args_list)
  local opts = parse_args(args_list)
  if opts.help then
    print(usage())
    return
  end
  if opts.invalid then
    util.err_write("invalid args:")
    for _, msg in ipairs(opts.invalid) do
      util.err_write("  " .. msg)
    end
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if opts.unknown then
    util.err_write("unknown args: " .. table.concat(opts.unknown, " "))
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  local root = vim.fs.normalize(vim.fn.fnamemodify(opts.root, ":p"))
  local cases_root = util.normalize_path(root, opts.cases_dir or "snapcase")
  local cases, errors = case_def.find_cases(cases_root)
  local filtered = case_def.filter_cases(cases, { tags = opts.tags, ids = opts.cases })

  if #errors > 0 then
    for _, msg in ipairs(errors) do
      util.err_write(msg)
    end
  end

  if opts.json then
    print_json(root, filtered)
  else
    print_text(filtered)
  end

  if #errors > 0 then
    vim.cmd("cquit 1")
  end
end

return M
