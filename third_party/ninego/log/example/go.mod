module ninego/log/logger/example

go 1.24.0

require go.uber.org/zap v1.27.0

require go.uber.org/multierr v1.10.0 // indirect

require (
	ninego/filelog v0.0.0-00010101000000-000000000000
	ninego/log v0.0.0-00010101000000-000000000000
	ninego/log/example v0.0.0-00010101000000-000000000000
)

replace ninego/filelog => ../../filelog

replace ninego/log => ..
replace ninego/log/example => .
