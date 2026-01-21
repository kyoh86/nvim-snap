vim.pack.add({ "https://github.com/kyoh86/qlean.nvim" }, { confirm = false })

local rule = require("qlean.rule")
require("qlean").setup({
  keep = rule.any(rule.buftype("", "acwrite", "terminal"), rule.filetype("fern")),
})

vim.fn.setline(1, { "test for qlean quickfix" })
vim.cmd.copen()
vim.cmd.new({ mods = { split = "topleft" } })
pcall(vim.api.nvim_cmd, { cmd = "quit" }, { output = true })
vim.cmd.redraw()
require("nvim_snap").done()
