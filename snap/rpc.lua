local uv = vim.loop

local M = {}

function M.start_embedded_nvim(opts)
  local env = {}
  for key, value in pairs(uv.os_environ()) do
    if key ~= "XDG_DATA_HOME" and key ~= "XDG_CONFIG_HOME" then
      env[#env + 1] = key .. "=" .. value
    end
  end
  if opts.data_home then
    env[#env + 1] = "XDG_DATA_HOME=" .. opts.data_home
  end
  if opts.config_home then
    env[#env + 1] = "XDG_CONFIG_HOME=" .. opts.config_home
  end
  if opts.log_file then
    local log_dir = vim.fn.fnamemodify(opts.log_file, ":h")
    if log_dir and log_dir ~= "." then
      vim.fn.mkdir(log_dir, "p")
    end
    local fd = io.open(opts.log_file, "a")
    if fd then
      fd:close()
    end
    env[#env + 1] = "NVIM_LOG_FILE=" .. opts.log_file
  end
  if opts.log_level then
    env[#env + 1] = "NVIM_LOG_LEVEL=" .. opts.log_level
  end

  local stdin = uv.new_pipe(false)
  local stdout = uv.new_pipe(false)
  local stderr = uv.new_pipe(false)
  local state = { exited = false, exit_code = nil, exit_signal = nil }
  local args = { "--embed", "--headless", "-u", "NONE", "-i", "NONE", "-n" }
  local handle, pid = uv.spawn(opts.nvim, {
    args = args,
    env = env,
    stdio = { stdin, stdout, stderr },
  }, function(code, signal)
    state.exited = true
    state.exit_code = code
    state.exit_signal = signal
  end)
  if not handle then
    return nil, string.format("failed to spawn %s", opts.nvim)
  end
  return {
    handle = handle,
    pid = pid,
    stdin = stdin,
    stdout = stdout,
    stderr = stderr,
    state = state,
  }
end

function M.new_rpc_client(proc, opts)
  local client = {
    proc = proc,
    msgid = 0,
    buffer = "",
    unpacker = vim.mpack.Unpacker(),
    responses = {},
    rpc_timeout = opts.rpc_timeout,
    on_notification = function(_, _) end,
    stderr_chunks = {},
  }

  local function handle_message(msg)
    local kind = msg[1]
    if kind == 1 then
      local id = msg[2]
      client.responses[id] = { err = msg[3], result = msg[4] }
    elseif kind == 2 then
      local method = msg[2]
      local params = msg[3] or {}
      client.on_notification(method, params)
    elseif kind == 0 then
      local id = msg[2]
      client:send({ 1, id, "request not supported", vim.NIL })
    end
  end

  local function feed(chunk)
    if not chunk then
      return
    end
    client.buffer = client.buffer .. chunk
    local pos = 1
    while pos <= #client.buffer do
      local ok, msg, next_pos = pcall(client.unpacker, client.buffer, pos)
      if not ok then
        break
      end
      pos = next_pos
      handle_message(msg)
    end
    if pos > 1 then
      client.buffer = string.sub(client.buffer, pos)
    end
  end

  proc.stdout:read_start(function(err, chunk)
    if err then
      client.last_error = err
      return
    end
    feed(chunk)
  end)

  proc.stderr:read_start(function(_, chunk)
    if chunk then
      table.insert(client.stderr_chunks, chunk)
    end
  end)

  function client:send(msg)
    local ok, err = pcall(function()
      self.proc.stdin:write(vim.mpack.encode(msg))
    end)
    if not ok then
      return false, err
    end
    return true
  end

  local function stderr_summary()
    if #client.stderr_chunks == 0 then
      return nil
    end
    local text = table.concat(client.stderr_chunks)
    if #text > 2000 then
      text = "...(truncated)...\n" .. text:sub(-2000)
    end
    text = text:gsub("%s+$", "")
    if text == "" then
      return nil
    end
    return text
  end

  function client:request(method, params)
    self.msgid = self.msgid + 1
    local id = self.msgid
    local ok, err = self:send({ 0, id, method, params or {} })
    if not ok then
      return nil, err
    end
    local done = vim.wait(self.rpc_timeout, function()
      return self.responses[id] ~= nil or self.proc.state.exited
    end, 5)
    if not done then
      local stderr = stderr_summary()
      if stderr then
        return nil, "rpc timeout\nstderr:\n" .. stderr
      end
      return nil, "rpc timeout"
    end
    if self.proc.state.exited then
      local stderr = stderr_summary()
      if stderr then
        return nil, "nvim exited\nstderr:\n" .. stderr
      end
      return nil, "nvim exited"
    end
    local resp = self.responses[id]
    self.responses[id] = nil
    if resp.err ~= nil and resp.err ~= vim.NIL then
      return nil, resp.err
    end
    return resp.result
  end

  function client:notify(method, params)
    return self:send({ 2, method, params or {} })
  end

  return client
end

return M
