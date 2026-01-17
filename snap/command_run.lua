local case_def = require("snap.case_def")
local output = require("snap.output")
local render = require("snap.render")
local snapshot = require("snap.snapshot")
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua run [options]",
    "",
    "options:",
    "  --root PATH       Root directory to search (default: snapcase)",
    "  --tag TAG         Filter by tag (repeatable, comma-separated)",
    "  --case ID         Filter by case id (repeatable, comma-separated)",
    "  --format FMT      Output formats: json,ansi,html (default: json)",
    "  -h, --help        Show this help",
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

local function parse_format(value)
  local formats = {}
  for _, item in ipairs(split_values(value)) do
    formats[item] = true
  end
  if not next(formats) then
    formats.json = true
  end
  return formats
end

local function parse_args(args)
  local opts = {
    root = "snapcase",
    tags = {},
    cases = {},
    formats = { json = true },
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
    elseif arg == "--format" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--format requires a value")
      else
        opts.formats = parse_format(value)
        i = i + 1
      end
    elseif vim.startswith(arg, "--format=") then
      opts.formats = parse_format(string.sub(arg, 10))
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

local function scenario_for_case(c)
  if c.kind == "regression" then
    return c.scenario_path
  end
  return c.target_scenario_path
end

local function collect_snapshot(c)
  local scenario = scenario_for_case(c)
  if vim.fn.filereadable(scenario) ~= 1 then
    return nil, "scenario not found: " .. scenario
  end
  local opts = {
    scripts = { scenario },
    width = 80,
    height = 24,
    wait = 200,
    rpc_timeout = 2000,
    nvim = "nvim",
    data_home = util.normalize_path(c.dir, ".nvim-data"),
    config_home = util.normalize_path(c.dir, ".nvim-config"),
    multigrid = false,
  }
  return snapshot.collect(opts)
end

local function write_outputs(c, formats, snap)
  local ok, err = case_def.ensure_dir(c.actual_dir)
  if not ok then
    return nil, err
  end
  if formats.json then
    local encoded = vim.json.encode(snap)
    local ok_write, write_err = output.write(c.actual_path, encoded)
    if not ok_write then
      return nil, write_err or "failed to write snapshot.json"
    end
  end
  if formats.ansi then
    local ansi = render.render_ansi(snap)
    local ok_write, write_err = output.write(vim.fs.joinpath(c.actual_dir, "snapshot.ansi"), ansi)
    if not ok_write then
      return nil, write_err or "failed to write snapshot.ansi"
    end
  end
  if formats.html then
    local html = render.render_html(snap)
    local ok_write, write_err = output.write(vim.fs.joinpath(c.actual_dir, "snapshot.html"), html)
    if not ok_write then
      return nil, write_err or "failed to write snapshot.html"
    end
  end
  return true
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
  local failed = #errors > 0

  for _, c in ipairs(filtered) do
    local snap, err = collect_snapshot(c)
    if not snap then
      util.err_write(c.id .. ": " .. (err or "capture failed"))
      failed = true
    else
      local ok_write, write_err = write_outputs(c, opts.formats, snap)
      if not ok_write then
        util.err_write(c.id .. ": " .. (write_err or "failed to write outputs"))
        failed = true
      end
    end
  end

  if failed then
    vim.cmd("cquit 1")
  end
end

return M
