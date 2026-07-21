export interface ProcessingConfig {
  source_dir: string
  output_dir: string
  record_step: number
  target_step: number
  num_threads: number
}

export type LogLevel = 'info' | 'skip' | 'error' | 'pending'

export interface LogEntry {
  level: LogLevel
  message: string
}

export interface ProgressPayload {
  current: number
  total: number
}

export interface LogPayload {
  level: LogLevel
  message: string
}

export interface ProcessingDonePayload {
  total_clips: number
  skipped: number
  errors: number
}
