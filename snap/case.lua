local util = require("snap.util")

local M = {}

local function normalize_rtp(case_dir, rtp)
  if type(rtp) == "string" then
    rtp = { rtp }
  end
  if type(rtp) ~= "table" then
    return {}
  end
  local out = {}
  for _, value in ipairs(rtp) do
    if type(value) == "string" and value ~= "" then
      table.insert(out, util.normalize_path(case_dir, value))
    end
  end
  return out
end

function M.load(opts)
  local case_path = opts.case
  if type(case_path) ~= "string" or case_path == "" then
    return nil, "case path is required"
  end
  local case_dir = vim.fs.normalize(vim.fn.fnamemodify(case_path, ":p"))
  if vim.fn.isdirectory(case_dir) ~= 1 then
    return nil, string.format("case dir not found: %s", case_dir)
  end
  local config_path = util.joinpath(case_dir, "snapcase.json")
  local config = {}
  if vim.fn.filereadable(config_path) == 1 then
    local fd, err = io.open(config_path, "r")
    if not fd then
      return nil, err
    end
    local text = fd:read("*a")
    fd:close()
    local ok, decoded = pcall(vim.json.decode, text)
    if not ok then
      return nil, string.format("failed to parse snapcase.json: %s", decoded)
    end
    if type(decoded) == "table" then
      config = decoded
    end
  end

  local scenario = config.scenario
  if type(scenario) ~= "string" then
    scenario = "scenario.lua"
  end
  ---@cast scenario string
  local scenario_path = util.normalize_path(case_dir, scenario)
  if not scenario_path or vim.fn.filereadable(scenario_path) ~= 1 then
    return nil, string.format("scenario not found: %s", scenario_path or scenario)
  end

  if type(config.width) == "number" then
    opts.width = config.width
  end
  if type(config.height) == "number" then
    opts.height = config.height
  end

  local out_dir = config.out_dir or ".out"
  local out_dir_path = util.normalize_path(case_dir, out_dir)
  local outputs = config.outputs or {}
  local default_outputs = {
    json = "snapshot.json",
    ansi = "snapshot.ansi",
    html = "snapshot.html",
  }
  local function resolve_output(key)
    local value = outputs[key]
    if value == nil then
      value = default_outputs[key]
    end
    if value == false or value == "none" then
      return nil
    end
    if value == "-" then
      return "-"
    end
    if type(value) == "string" then
      return util.normalize_path(out_dir_path, value)
    end
    return nil
  end

  opts.json_out = resolve_output("json")
  opts.ansi_out = resolve_output("ansi")
  opts.html_out = resolve_output("html")
  opts.data_home = util.normalize_path(case_dir, config.data_home or ".nvim-data")
  opts.config_home = util.normalize_path(case_dir, config.config_home or ".nvim-config")
  opts.rtp = normalize_rtp(case_dir, config.rtp)
  opts.scripts = { scenario_path }

  return opts
end

return M
