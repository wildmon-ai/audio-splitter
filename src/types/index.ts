export interface ProcessingConfig {
  source_dir: string
  output_dir: string
  record_step: number
  target_step: number
  num_threads: number
}

export type LogLevel = 'info' | 'warn' | 'error'

export interface LogPayload {
  level: LogLevel
  message: string
}

export interface LogEntry extends LogPayload {
  id: number
}

/** 'scan' while the durations are read, then 'clips' while clips are cut. */
export type ProgressPhase = 'scan' | 'clips'

export interface ProgressPayload {
  phase?: ProgressPhase
  current: number
  total: number
}

export interface ProcessingDonePayload {
  total_clips: number
  skipped: number
  errors: number
  /** The job was cancelled before it finished. */
  cancelled?: boolean
  /** The job could not run at all (missing folder, ffmpeg not found…). */
  error?: string
}
