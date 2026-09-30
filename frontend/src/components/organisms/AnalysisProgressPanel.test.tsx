import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import type { Analysis } from '../../api/types'
import { routerFuture } from '../../app/router'
import { StageStepper, STAGES } from '../molecules/StageStepper'
import { AnalysisProgressPanel } from './AnalysisProgressPanel'

const analysis = (over: Partial<Analysis> = {}): Analysis => ({
  id: 'a-1',
  status: 'running',
  stage: 'detection',
  stage_index: 3,
  stage_total: 7,
  progress: 28,
  created_at: '2026-09-30T10:00:00Z',
  started_at: '2026-09-30T10:00:01Z',
  finished_at: null,
  error: '',
  summary: null,
  ...over,
})

const done = (text: string) =>
  analysis({
    status: 'done',
    stage: 'recommendation',
    stage_index: 7,
    progress: 100,
    finished_at: '2026-09-30T10:00:05Z',
    summary: { total: 4, priority_attention: 2, meters_analyzed: 12, by_type: {}, by_severity: {}, avg_confidence: 0.9, text },
  })

function renderPanel(a: Analysis, onRetry = vi.fn()) {
  render(
    <MemoryRouter future={routerFuture}>
      <AnalysisProgressPanel analysis={a} onRetry={onRetry} />
    </MemoryRouter>,
  )
  return { onRetry }
}

const steps = () => within(screen.getByRole('list', { name: 'Analysis stages' })).getAllByRole('listitem')

describe('StageStepper', () => {
  it('names the seven stages in pipeline order', () => {
    render(<StageStepper stageIndex={0} status="pending" />)
    expect(steps().map((s) => s.textContent)).toEqual(
      expect.arrayContaining(['Readings', 'Baseline', 'Detection', 'Correlation', 'Events', 'Explanation', 'Recommendation'].map((n) => expect.stringContaining(n))),
    )
    expect(STAGES).toHaveLength(7)
  })

  it('marks earlier stages done, the current one in progress and the rest pending', () => {
    render(<StageStepper stageIndex={3} status="running" />)
    const items = steps()
    expect(items[0]).toHaveTextContent('Readings (done)')
    expect(items[1]).toHaveTextContent('Baseline (done)')
    expect(items[2]).toHaveTextContent('Detection (in progress)')
    expect(items[2]).toHaveAttribute('aria-current', 'step')
    expect(items[3]).toHaveTextContent('Correlation (pending)')
    expect(items[6]).toHaveTextContent('Recommendation (pending)')
    expect(items.filter((i) => i.hasAttribute('aria-current'))).toHaveLength(1)
  })

  it('completes every stage when the run is done', () => {
    render(<StageStepper stageIndex={7} status="done" />)
    for (const item of steps()) expect(item).toHaveTextContent('(done)')
    expect(steps().some((i) => i.hasAttribute('aria-current'))).toBe(false)
  })

  it('flags the stage that failed', () => {
    render(<StageStepper stageIndex={4} status="failed" />)
    expect(steps()[3]).toHaveTextContent('Correlation (failed)')
    expect(steps()[2]).toHaveTextContent('(done)')
    expect(steps()[4]).toHaveTextContent('(pending)')
  })
})

describe('AnalysisProgressPanel while running', () => {
  it('shows the status, progress bar, stepper and a polite announcement', () => {
    renderPanel(analysis())

    expect(screen.getByText('Running')).toBeInTheDocument()
    expect(screen.getByRole('progressbar', { name: 'Analysis progress' })).toHaveAttribute('aria-valuenow', '28')
    expect(screen.getByText('28%')).toBeInTheDocument()
    expect(steps()[2]).toHaveTextContent('Detection (in progress)')
    const live = screen.getByRole('status')
    expect(live).toHaveAttribute('aria-live', 'polite')
    expect(live).toHaveTextContent('Stage 3 of 7: Detection')
  })

  it('follows the run as it advances', () => {
    renderPanel(analysis({ stage: 'explanation', stage_index: 6, progress: 71 }))

    expect(screen.getByRole('progressbar', { name: 'Analysis progress' })).toHaveAttribute('aria-valuenow', '71')
    expect(screen.getByRole('status')).toHaveTextContent('Stage 6 of 7: Explanation')
  })

  it('shows a pending run as waiting to start', () => {
    renderPanel(analysis({ status: 'pending', stage: '', stage_index: 0, progress: 0 }))

    expect(screen.getByText('Pending')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('Waiting to start')
    expect(steps().every((i) => i.textContent?.includes('(pending)'))).toBe(true)
  })

  it('offers no link to anomalies before the run is done', () => {
    renderPanel(analysis())
    expect(screen.queryByRole('link', { name: 'View anomalies' })).not.toBeInTheDocument()
  })
})

describe('AnalysisProgressPanel when done', () => {
  it('shows the headline from the API and a link to the anomalies', () => {
    renderPanel(done('4 anomalies detected · 2 require priority attention'))

    expect(screen.getByText('Done')).toBeInTheDocument()
    expect(screen.getByText('4 anomalies detected · 2 require priority attention')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'View anomalies' })).toHaveAttribute('href', '/anomalies')
    expect(screen.getByRole('progressbar', { name: 'Analysis progress' })).toHaveAttribute('aria-valuenow', '100')
  })

  it('never hardcodes the headline', () => {
    renderPanel(done('No anomalies detected'))

    expect(screen.getByText('No anomalies detected')).toBeInTheDocument()
    expect(screen.queryByText(/4 anomalies/)).not.toBeInTheDocument()
  })
})

describe('AnalysisProgressPanel when failed', () => {
  it('announces the error and retries on demand', async () => {
    const { onRetry } = renderPanel(analysis({ status: 'failed', error: 'engine crashed' }))

    expect(screen.getByText('Failed')).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('engine crashed')
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('falls back to a generic message when the run gave no reason', () => {
    renderPanel(analysis({ status: 'failed', error: '' }))
    expect(screen.getByRole('alert')).toHaveTextContent('The analysis failed')
  })
})
