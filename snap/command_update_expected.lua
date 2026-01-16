local case_def = require("snap.case_def")
local output = require("snap.output")
local snapshot = require("snap.snapshot")
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua update-expected [options]",
    "",
    "options:",
    "  --root PATH       Root directory to search (default: .)",
    "  --tag TAG         Filter by tag (repeatable, comma-separated)",
    "  --case ID         Filter by case id (repeatable, comma-separated)",
    "  --dry-run         Show updates without writing",
    "  --no-confirm      Skip confirmation prompt",
    "  --yes             Alias of --no-confirm",
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

local function parse_args(args)
  local opts = {
    root = ".",
    tags = {},
    cases = {},
    dry_run = false,
    confirm = true,
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
    elseif arg == "--dry-run" then
      opts.dry_run = true
    elseif arg == "--no-confirm" or arg == "--yes" then
      opts.confirm = false
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

local function read_input(path)
  local fd, err = io.open(path, "r")
  if not fd then
    return nil, err
  end
  local text = fd:read("*a")
  fd:close()
  return text
end

local function build_action(c)
  if c.kind == "regression" then
    if vim.fn.filereadable(c.actual_path) ~= 1 then
      return nil, "actual not found: " .. c.actual_path
    end
    return {
      case = c,
      kind = "copy",
      src = c.actual_path,
      dst = c.expected_path,
    }
  end
  if vim.fn.filereadable(c.golden_scenario_path) ~= 1 then
    return nil, "golden scenario not found: " .. c.golden_scenario_path
  end
  return {
    case = c,
    kind = "capture",
    scenario = c.golden_scenario_path,
    dst = c.expected_path,
  }
end

local function collect_snapshot(c, scenario)
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

local function describe_action(action)
  if action.kind == "copy" then
    return action.src .. " -> " .. action.dst
  end
  return action.scenario .. " -> " .. action.dst
end

local function print_actions(actions)
  for _, action in ipairs(actions) do
    print(action.case.id .. "\t" .. describe_action(action))
  end
end

local function confirm_actions(actions)
  local message = string.format("Update expected snapshots for %d case(s)?", #actions)
  local choice = vim.fn.confirm(message, "&Yes\n&No", 2)
  return choice == 1
end

local function perform_action(action)
  local c = action.case
  local ok, err = case_def.ensure_dir(c.expected_dir)
  if not ok then
    return nil, err
  end
  if action.kind == "copy" then
    local text, read_err = read_input(action.src)
    if not text then
      return nil, read_err or "failed to read actual"
    end
    local ok_write, write_err = output.write(action.dst, text)
    if not ok_write then
      return nil, write_err or "failed to write expected"
    end
    return true
  end
  local snap, err_snap = collect_snapshot(c, action.scenario)
  if not snap then
    return nil, err_snap or "capture failed"
  end
  local encoded = vim.json.encode(snap)
  local ok_write, write_err = output.write(action.dst, encoded)
  if not ok_write then
    return nil, write_err or "failed to write expected"
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
  local actions = {}
  local failed = #errors > 0

  for _, err in ipairs(errors) do
    util.err_write(err)
  end

  for _, c in ipairs(filtered) do
    local action, err = build_action(c)
    if not action then
      util.err_write(c.id .. ": " .. (err or "failed to build action"))
      failed = true
    else
      table.insert(actions, action)
    end
  end

  if #actions == 0 then
    if failed then
      vim.cmd("cquit 1")
    end
    return
  end

  if opts.dry_run then
    print_actions(actions)
    if failed then
      vim.cmd("cquit 1")
    end
    return
  end

  if opts.confirm and not confirm_actions(actions) then
    util.err_write("aborted")
    vim.cmd("cquit 1")
    return
  end

  for _, action in ipairs(actions) do
    local ok, err = perform_action(action)
    if not ok then
      util.err_write(action.case.id .. ": " .. (err or "failed to update expected"))
      failed = true
    else
      print(action.case.id .. "\tupdated")
    end
  end

  if failed then
    vim.cmd("cquit 1")
  end
end

return M
