local M = {}

local function parse_size()
  local value = vim.env.SNAP_PNG_SIZE
  if type(value) ~= "string" or value == "" then
    return 1600, 1200
  end
  local w, h = value:match("^(%d+)[xX](%d+)$")
  w = tonumber(w)
  h = tonumber(h)
  if not w or not h or w <= 0 or h <= 0 then
    return 1600, 1200
  end
  return w, h
end

local function find_tool()
  local candidates = {
    "google-chrome",
    "chrome",
    "msedge",
    "chromium",
    "chromium-browser",
    "wkhtmltoimage",
  }
  for _, cmd in ipairs(candidates) do
    if vim.fn.executable(cmd) == 1 then
      return cmd
    end
  end
  return nil
end

local function write_file(path, contents)
  local fd, err = io.open(path, "w")
  if not fd then
    return nil, err
  end
  fd:write(contents)
  fd:close()
  return true
end

local function run_chromium(cmd, html_path, out_path, width, height, user_data_dir)
  local url = vim.uri_from_fname(html_path)
  local profile_dir = vim.fs.joinpath(user_data_dir, "profile")
  vim.fn.mkdir(profile_dir, "p")
  local args = {
    "env",
    "HOME=" .. user_data_dir,
    "XDG_DATA_HOME=" .. user_data_dir,
    "XDG_CONFIG_HOME=" .. user_data_dir,
    cmd,
    "--headless",
    "--disable-gpu",
    "--no-sandbox",
    "--no-first-run",
    "--disable-extensions",
    "--disable-dev-shm-usage",
    "--disable-crash-reporter",
    "--disable-breakpad",
    "--crash-dumps-dir=" .. user_data_dir,
    "--disable-features=Translate,Crashpad",
    "--user-data-dir=" .. profile_dir,
    "--window-size=" .. width .. "," .. height,
    "--screenshot=" .. out_path,
    url,
  }
  local output = vim.fn.system(args)
  if vim.v.shell_error ~= 0 then
    return nil, output
  end
  return true
end

local function run_wkhtmltoimage(cmd, html_path, out_path, width)
  local args = {
    cmd,
    "--width",
    tostring(width),
    "--disable-smart-width",
    html_path,
    out_path,
  }
  local output = vim.fn.system(args)
  if vim.v.shell_error ~= 0 then
    return nil, output
  end
  return true
end

---@param html string
---@param out_path string
---@return boolean|nil
---@return string|nil
function M.write_png_from_html(html, out_path)
  local tool = find_tool()
  if not tool then
    return nil, "png tool not found (chromium/chrome/msedge/wkhtmltoimage)"
  end
  local width, height = parse_size()
  local tmp = vim.fn.tempname() .. ".html"
  local user_data_dir = vim.fn.tempname()
  local ok, err = write_file(tmp, html)
  if not ok then
    return nil, err
  end
  if tool ~= "wkhtmltoimage" then
    vim.fn.mkdir(user_data_dir, "p")
  end
  local ok_run, run_err
  if tool == "wkhtmltoimage" then
    ok_run, run_err = run_wkhtmltoimage(tool, tmp, out_path, width)
  else
    ok_run, run_err = run_chromium(tool, tmp, out_path, width, height, user_data_dir)
  end
  os.remove(tmp)
  if tool ~= "wkhtmltoimage" then
    vim.fn.delete(user_data_dir, "rf")
  end
  if not ok_run then
    return nil, run_err
  end
  return true
end

return M
