import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { routerFuture } from '../../app/router'
import { demoAnomalies, doneAnalysis, makeAnalysis } from '../../test/fixtures'
import { AnomaliesView, type AnomaliesViewProps } from './AnomaliesView'

function setup(overrides: Partial<AnomaliesViewProps> = {}) {
  const props: AnomaliesViewProps = {
    state: 'ready',
    list: { analysis_id: 'a-1', summary_text: '4 anomalies detected · 2 require priority attention', items: demoAnomalies() },
    filters: { type: '', severity: '', meterId: '' },
    onTypeChange: vi.fn(),
    onSeverityChange: vi.fn(),
    onClearFilters: vi.fn(),
    onRetry: vi.fn(),
    onRun: vi.fn(),
    starting: false,
    busy: false,
    runError: null,
    ...overrides,
  }
  render(
    <MemoryRouter future={routerFuture}>
      <AnomaliesView {...props} />
    </MemoryRouter>,
  )
  return props
}

describe('AnomaliesView', () => {
  it('shows the summary banner and the ranked table', () => {
    setup()
    expect(screen.getByRole('heading', { level: 1, name: 'AI Anomalies' })).toBeInTheDocument()
    expect(screen.getByText('4 anomalies detected · 2 require priority attention')).toBeInTheDocument()
    expect(screen.getAllByRole('row')).toHaveLength(5)
  })

  it('reports filter changes for type and severity', async () => {
    const props = setup()
    await userEvent.selectOptions(screen.getByLabelText('Type'), 'DATA_QUALITY')
    await userEvent.selectOptions(screen.getByLabelText('Severity'), 'HIGH')
    expect(props.onTypeChange).toHaveBeenCalledWith('DATA_QUALITY')
    expect(props.onSeverityChange).toHaveBeenCalledWith('HIGH')
  })

  it('reflects the active filters in the controls', () => {
    setup({ filters: { type: 'REAL_ANOMALY', severity: 'HIGH', meterId: '' } })
    expect(screen.getByLabelText('Type')).toHaveValue('REAL_ANOMALY')
    expect(screen.getByLabelText('Severity')).toHaveValue('HIGH')
  })

  it('names the meter filter and clears it', async () => {
    const props = setup({ filters: { type: '', severity: '', meterId: 'M-109' } })
    expect(screen.getByText('Meter M-109')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Clear filters' }))
    expect(props.onClearFilters).toHaveBeenCalled()
  })

  it('offers to run the analysis when none has completed', async () => {
    const props = setup({ list: { analysis_id: null, summary_text: null, items: [] } })
    expect(screen.getByText('No analysis yet')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Run AI Analysis' }))
    expect(props.onRun).toHaveBeenCalled()
  })

  it('says when the filters match nothing and offers to clear them', async () => {
    const props = setup({ list: { analysis_id: 'a-1', summary_text: 'x', items: [] }, filters: { type: 'FALSE_POSITIVE', severity: 'HIGH', meterId: '' } })
    expect(screen.getByText('No anomalies match these filters')).toBeInTheDocument()
    await userEvent.click(screen.getAllByRole('button', { name: 'Clear filters' })[0]!)
    expect(props.onClearFilters).toHaveBeenCalled()
  })

  it('says when the analysis found nothing', () => {
    setup({ list: { analysis_id: 'a-1', summary_text: 'No anomalies detected', items: [] } })
    expect(screen.getByText('No anomalies were detected')).toBeInTheDocument()
  })

  it('shows loading and error states', async () => {
    const props = setup({ state: 'loading', list: undefined })
    expect(screen.getByRole('status', { name: 'Loading anomalies' })).toBeInTheDocument()
    expect(props.onRetry).not.toHaveBeenCalled()
  })

  it('announces an error and retries', async () => {
    const props = setup({ state: 'error', list: undefined, errorMessage: 'Cannot reach the server.' })
    expect(screen.getByRole('alert')).toHaveTextContent('Cannot reach the server.')
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(props.onRetry).toHaveBeenCalled()
  })

  it('shows the live progress of a run started here and disables the trigger', () => {
    setup({ busy: true, analysis: makeAnalysis(), list: { analysis_id: 'a-1', summary_text: 'x', items: demoAnomalies() } })
    expect(screen.getByRole('region', { name: 'AI analysis' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Analyzing/ })).toBeDisabled()
  })

  it('links to the anomalies from a finished run without repeating the banner', () => {
    setup({ analysis: doneAnalysis() })
    expect(screen.getAllByText('4 anomalies detected · 2 require priority attention')).toHaveLength(1)
  })
})
