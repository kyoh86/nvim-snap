local case_def = require("snap.case_def")
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua list [options]",
    "",
    "options:",
    "  --root PATH    Root directory to search (default: .)",
    "  --tag TAG      Filter by tag (repeatable, comma-separated)",
    "  --case ID      Filter by case id (repeatable, comma-separated)",
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
  for _, c in ipairs(cases) do
    local tags = table.concat(c.tags, ",")
    print(table.concat({ c.id, c.name, c.kind, tags, c.path }, "\t"))
  end
end

local function print_json(root, cases)
  local out = {
    root = root,
    cases = {},
  }
  for _, c in ipairs(cases) do
    table.insert(out.cases, {
      id = c.id,
      name = c.name,
      kind = c.kind,
      tags = c.tags,
      path = c.path,
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
  local cases, errors = case_def.find_cases(root)
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
