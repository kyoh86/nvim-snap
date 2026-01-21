vim.pack.add({ "https://github.com/kyoh86/qlean.nvim" }, { confirm = false })

local rule = require("qlean.rule")
require("qlean").setup({
  keep = rule.any(rule.buftype("", "acwrite", "terminal"), rule.filetype("fern")),
})

vim.go.hidden = true

vim.cmd.new()
vim.fn.setline(1, { "foobar" })
vim.cmd.wincmd("c")

vim.cmd.copen()
vim.cmd.wincmd("k")
pcall(vim.api.nvim_cmd, { cmd = "quit" }, { output = true })
vim.cmd.redraw()
require("nvim_snap").done()
