local util = require("snap.util")

local M = {}

---@class SnapCase
---@field name string
---@field title string
---@field kind "regression"|"golden"
---@field tags string[]
---@field dir string
---@field path string
---@field expected_dir string
---@field actual_dir string
---@field diff_dir string
---@field expected_path string
---@field actual_path string
---@field scenario_path string
---@field golden_scenario_path string
---@field target_scenario_path string
---@field width integer|nil
---@field height integer|nil
---@field wait integer|nil
---@field rpc_timeout integer|nil
---@field data_home string
---@field config_home string
---@field log_file string|nil
---@field rtp string[]

local function read_json(path)
  local fd, err = io.open(path, "r")
  if not fd then
    return nil, err
  end
  local text = fd:read("*a")
  fd:close()
  local ok, decoded = pcall(vim.json.decode, text)
  if not ok then
    return nil, "failed to parse snapcase.json: " .. tostring(decoded)
  end
  if type(decoded) ~= "table" then
    return nil, "invalid snapcase.json: expected object"
  end
  return decoded
end

local function basename(path)
  return vim.fs.basename(vim.fs.normalize(path))
end

local function expand_placeholders(value, vars)
  return (value:gsub("%${([A-Z_]+)}", function(key)
    return vars[key] or ""
  end))
end

local function is_absolute(path)
  if path:sub(1, 1) == "/" then
    return true
  end
  return path:match("^%a:[/\\]") ~= nil
end

local function normalize_rtp(case_dir, root, rtp)
  if type(rtp) == "string" then
    rtp = { rtp }
  end
  if type(rtp) ~= "table" then
    return {}
  end
  local out = {}
  local vars = {
    CASE = case_dir,
    ROOT = root or case_dir,
  }
  for _, value in ipairs(rtp) do
    if type(value) == "string" and value ~= "" then
      local expanded = expand_placeholders(value, vars)
      if expanded ~= "" then
        if is_absolute(expanded) then
          table.insert(out, vim.fs.normalize(expanded))
        else
          table.insert(out, util.normalize_path(case_dir, expanded))
        end
      end
    end
  end
  return out
end

local function normalize_tags(tags)
  if type(tags) ~= "table" then
    return {}
  end
  local out = {}
  for _, tag in ipairs(tags) do
    if type(tag) == "string" and tag ~= "" then
      table.insert(out, tag)
    end
  end
  return out
end

---@param case_path string
---@return SnapCase|nil
---@return string|nil
function M.load_case(case_path, root)
  local case_dir = vim.fs.normalize(vim.fn.fnamemodify(case_path, ":p:h"))
  local config, err = read_json(case_path)
  if not config then
    return nil, err
  end
  local version = config.version
  if type(version) ~= "number" or version < 1 or math.floor(version) ~= version then
    return nil, "case version is required"
  end
  local name = basename(case_dir)
  if type(name) ~= "string" or name == "" then
    return nil, "case dir name is required"
  end
  local kind = config.kind
  if kind ~= "regression" and kind ~= "golden" then
    return nil, "case kind must be regression or golden"
  end
  local title = config.title
  if type(title) ~= "string" or title == "" then
    title = name
  end
  local tags = normalize_tags(config.tags)
  local width = config.width
  if type(width) ~= "number" or width <= 0 then
    width = nil
  end
  local height = config.height
  if type(height) ~= "number" or height <= 0 then
    height = nil
  end
  local wait = config.wait
  if type(wait) ~= "number" or wait <= 0 then
    wait = nil
  end
  local rpc_timeout = config.rpc_timeout
  if type(rpc_timeout) ~= "number" or rpc_timeout <= 0 then
    rpc_timeout = nil
  end
  local log_file = config.log_file
  if type(log_file) ~= "string" or log_file == "" then
    log_file = nil
  end
  local data_home = util.normalize_path(case_dir, config.data_home or ".nvim-data")
  local config_home = util.normalize_path(case_dir, config.config_home or ".nvim-config")
  local rtp = normalize_rtp(case_dir, root or case_dir, config.rtp)

  local expected_dir = util.normalize_path(case_dir, "expected")
  local actual_dir = util.normalize_path(case_dir, "actual")
  local diff_dir = util.normalize_path(case_dir, "diff")

  local scenario = util.normalize_path(case_dir, "scenario.lua")
  local golden = util.normalize_path(case_dir, "golden.lua")
  local target = util.normalize_path(case_dir, "target.lua")

  return {
    name = name,
    title = title,
    kind = kind,
    tags = tags,
    dir = case_dir,
    path = case_dir,
    expected_dir = expected_dir,
    actual_dir = actual_dir,
    diff_dir = diff_dir,
    expected_path = util.normalize_path(expected_dir, "snapshot.json"),
    actual_path = util.normalize_path(actual_dir, "snapshot.json"),
    scenario_path = scenario,
    golden_scenario_path = golden,
    target_scenario_path = target,
    width = width,
    height = height,
    wait = wait,
    rpc_timeout = rpc_timeout,
    data_home = data_home,
    config_home = config_home,
    log_file = log_file and util.normalize_path(case_dir, log_file) or nil,
    rtp = rtp,
  }
end

local function matches_tag(case_tags, filter_tags)
  if #filter_tags == 0 then
    return true
  end
  local tag_map = {}
  for _, tag in ipairs(case_tags) do
    tag_map[tag] = true
  end
  for _, tag in ipairs(filter_tags) do
    if tag_map[tag] then
      return true
    end
  end
  return false
end

local function matches_case_name(name, filter_names)
  if #filter_names == 0 then
    return true
  end
  for _, value in ipairs(filter_names) do
    if value == name then
      return true
    end
  end
  return false
end

function M.filter_cases(cases, filter)
  local out = {}
  for _, c in ipairs(cases) do
    if matches_tag(c.tags, filter.tags or {}) and matches_case_name(c.name, filter.ids or {}) then
      table.insert(out, c)
    end
  end
  return out
end

---@param root string
---@param cases_dir string|nil
---@return SnapCase[]
---@return string[]
function M.find_cases(root, cases_dir)
  local cases_root = util.normalize_path(root, cases_dir or "snapcase")
  local paths = vim.fn.globpath(cases_root, "*/snapcase.json", true, true)
  local cases = {}
  local errors = {}
  for _, path in ipairs(paths) do
    local c, err = M.load_case(path, root)
    if not c then
      table.insert(errors, string.format("%s: %s", path, err))
    else
      table.insert(cases, c)
    end
  end
  table.sort(cases, function(a, b)
    return a.name < b.name
  end)
  return cases, errors
end

function M.ensure_dir(path)
  if not path then
    return false, "path is required"
  end
  local ok = vim.fn.mkdir(path, "p")
  if ok == 0 then
    return false, "failed to create dir: " .. path
  end
  return true
end

return M
