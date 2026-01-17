local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim-snap new [options]",
    "",
    "options:",
    "  --root PATH       Root directory to create case (default: .)",
    "  --cases-dir PATH  Cases directory under root (default: snapcase)",
    "  --name NAME       Case name (optional, random if omitted)",
    "  --title TITLE     Case display title",
    "  --kind KIND       regression|golden (default: regression)",
    "  --tag TAG         Tag (repeatable, comma-separated)",
    "  --force           Overwrite existing files",
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
    cases_dir = "snapcase",
    name = nil,
    title = nil,
    kind = "regression",
    tags = {},
    force = false,
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
    elseif arg == "--name" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--name requires a value")
      else
        opts.name = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--name=") then
      opts.name = string.sub(arg, 8)
    elseif arg == "--title" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--title requires a value")
      else
        opts.title = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--title=") then
      opts.title = string.sub(arg, 9)
    elseif arg == "--kind" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--kind requires a value")
      else
        opts.kind = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--kind=") then
      opts.kind = string.sub(arg, 8)
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
    elseif arg == "--force" then
      opts.force = true
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

local function seed_random()
  local seed = os.time()
  if vim.loop and vim.loop.hrtime then
    local suffix = tostring(vim.loop.hrtime()):sub(-9)
    seed = tonumber(suffix) or seed
  end
  math.randomseed(seed)
  math.random()
  math.random()
  math.random()
end

---@param length number
---@return string
local function random_name(length)
  local chars = "abcdefghijklmnopqrstuvwxyz0123456789"
  local out = {}
  for i = 1, length do
    local idx = math.random(#chars)
    out[i] = chars:sub(idx, idx)
  end
  return table.concat(out)
end

---@param root string
---@param attempts integer|nil
---@return string|nil
---@return string|nil
local function generate_case_name(root, attempts)
  local count = attempts or 50
  for _ = 1, count do
    local name = random_name(8)
    local dir = vim.fs.joinpath(root, name)
    if vim.fn.isdirectory(dir) ~= 1 then
      return name
    end
  end
  return nil, "failed to generate unique case name"
end

local function file_exists(path)
  return vim.fn.filereadable(path) == 1
end

local function write_file(path, contents, force)
  if file_exists(path) and not force then
    return nil, "file exists: " .. path
  end
  local dir = vim.fn.fnamemodify(path, ":h")
  if dir and dir ~= "." then
    vim.fn.mkdir(dir, "p")
  end
  local fd, err = io.open(path, "w")
  if not fd then
    return nil, err
  end
  fd:write(contents)
  fd:close()
  return true
end

---@param path string
---@param force boolean
---@return boolean|nil
---@return string|nil
local function write_case_gitignore(path, force)
  local contents = table.concat({
    ".nvim-data/",
    ".nvim-config/",
    ".out/",
    "actual/",
    "diff/",
    "",
  }, "\n")
  return write_file(path, contents, force)
end

---@param path string
---@param force boolean
---@return boolean|nil
---@return string|nil
---@param path string
---@param opts table
---@return boolean|nil
---@return string|nil
local function write_snapcase_json(path, opts)
  local payload = {
    version = 1,
    kind = opts.kind,
    scenario = "scenario.lua",
    out_dir = ".out",
    data_home = ".nvim-data",
    config_home = ".nvim-config",
    outputs = {
      json = "snapshot.json",
      ansi = "snapshot.ansi",
      html = "snapshot.html",
    },
    rtp = {
      ".",
    },
  }
  if opts.title and opts.title ~= "" then
    payload.title = opts.title
  end
  if #opts.tags > 0 then
    payload.tags = opts.tags
  end
  local encoded = vim.json.encode(payload, { indent = "  " })
  return write_file(path, encoded, opts.force)
end

---@param path string
---@param name string
---@param force boolean
---@return boolean|nil
---@return string|nil
local function write_regression_scenario(path, name, force)
  local contents = table.concat({
    "vim.cmd(\"enew\")",
    "vim.fn.setline(1, {",
    "  \"case: " .. name .. "\",",
    "  \"edit this scenario\",",
    "})",
    "",
  }, "\n")
  return write_file(path, contents, force)
end

---@param path string
---@param name string
---@param label string
---@param force boolean
---@return boolean|nil
---@return string|nil
local function write_golden_scenario(path, name, label, force)
  local contents = table.concat({
    "vim.cmd(\"enew\")",
    "vim.fn.setline(1, {",
    "  \"" .. label .. " view for " .. name .. "\",",
    "  \"edit this scenario\",",
    "})",
    "",
  }, "\n")
  return write_file(path, contents, force)
end

---@param args_list string[]
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
  if opts.kind ~= "regression" and opts.kind ~= "golden" then
    util.err_write("--kind must be regression or golden")
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  seed_random()
  local root = vim.fs.normalize(vim.fn.fnamemodify(opts.root, ":p"))
  local cases_root = util.normalize_path(root, opts.cases_dir or "snapcase")
  vim.fn.mkdir(cases_root, "p")
  local case_dir
  if not opts.name or opts.name == "" then
    local generated, err = generate_case_name(cases_root)
    if not generated then
      util.err_write(err or "failed to generate case name")
      vim.cmd("cq")
      return
    end
    opts.name = generated
  end
  case_dir = vim.fs.normalize(vim.fs.joinpath(cases_root, opts.name))

  if vim.fn.isdirectory(case_dir) == 1 and not opts.force then
    util.err_write("case directory already exists: " .. case_dir)
    vim.cmd("cq")
    return
  end
  vim.fn.mkdir(case_dir, "p")

  local ok, err = write_snapcase_json(vim.fs.joinpath(case_dir, "snapcase.json"), opts)
  if not ok then
    util.err_write(err or "failed to write snapcase.json")
    vim.cmd("cq")
    return
  end
  local ok_gitignore, err_gitignore = write_case_gitignore(vim.fs.joinpath(case_dir, ".gitignore"), opts.force)
  if not ok_gitignore then
    util.err_write(err_gitignore or "failed to write .gitignore")
    vim.cmd("cq")
    return
  end
  vim.fn.mkdir(vim.fs.joinpath(case_dir, "expected"), "p")
  vim.fn.mkdir(vim.fs.joinpath(case_dir, "actual"), "p")
  vim.fn.mkdir(vim.fs.joinpath(case_dir, "diff"), "p")

  if opts.kind == "regression" then
    local ok_s, err_s = write_regression_scenario(vim.fs.joinpath(case_dir, "scenario.lua"), opts.name, opts.force)
    if not ok_s then
      util.err_write(err_s or "failed to write scenario.lua")
      vim.cmd("cq")
      return
    end
  else
    local ok_g, err_g = write_golden_scenario(vim.fs.joinpath(case_dir, "golden.lua"), opts.name, "golden", opts.force)
    if not ok_g then
      util.err_write(err_g or "failed to write golden.lua")
      vim.cmd("cq")
      return
    end
    local ok_t, err_t = write_golden_scenario(vim.fs.joinpath(case_dir, "target.lua"), opts.name, "target", opts.force)
    if not ok_t then
      util.err_write(err_t or "failed to write target.lua")
      vim.cmd("cq")
      return
    end
  end

  print(case_dir)
end

return M
