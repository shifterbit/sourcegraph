import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { assert, stub } from 'sinon'

import { LineChart } from './LineChart'
import { FLAT_SERIES } from './story/mocks'

const defaultArgs: RenderChartArgs = { series: FLAT_SERIES }

interface RenderChartArgs {
    series: typeof FLAT_SERIES
}
const renderChart = ({ series }: RenderChartArgs) => render(<LineChart width={400} height={400} series={series} />)

describe('LineChart', () => {
    // Non-exhaustive smoke tests to check that the chart renders correctly
    // All other general rendering tests are covered by chromatic
    describe('should render', () => {
        it('empty series', () => {
            renderChart({ ...defaultArgs, series: [] })
        })

        it('series with data', () => {
            renderChart(defaultArgs)

            // Query chart series list
            const series = screen.getByLabelText('Chart series')

            // Check that series were rendered
            const series1 = within(series).getByLabelText('A metric')
            const series2 = within(series).getByLabelText('C metric')
            const series3 = within(series).getByLabelText('B metric')
            expect(series1)
            expect(series2)
            expect(series3)

            // Check number of data points rendered
            expect(within(series1).getAllByRole('listitem')).toHaveLength(FLAT_SERIES[0].data.length)
            expect(within(series2).getAllByRole('listitem')).toHaveLength(FLAT_SERIES[1].data.length)
            expect(within(series3).getAllByRole('listitem')).toHaveLength(FLAT_SERIES[2].data.length)

            // Spot check y axis labels
            expect(screen.getByLabelText(/axis tick, value: 8/i)).toBeInTheDocument()
            expect(screen.getByLabelText(/axis tick, value: 20/i)).toBeInTheDocument()
            expect(screen.getByLabelText(/axis tick, value: 36/i)).toBeInTheDocument()

            // Spot check x axis labels
            expect(screen.getByLabelText(/axis tick, value: .*jan 01 2021/i)).toBeInTheDocument()
            expect(screen.getByLabelText(/axis tick, value: .*oct 01 2021/i)).toBeInTheDocument()
            expect(screen.getByLabelText(/axis tick, value: .*oct 01 2022/i)).toBeInTheDocument()
        })
    })

    describe('should handle clicks', () => {
        it('on a point', () => {
            const openStub = stub(window, 'open')

            renderChart(defaultArgs)

            // Query chart series list
            const series = screen.getByLabelText('Chart series')
            const [firstPoint, secondPoint, thirdPoint] = within(series).getAllByRole('listitem')

            // Spot checking multiple points
            // related issue https://github.com/sourcegraph/sourcegraph/issues/38304
            userEvent.click(firstPoint)
            userEvent.click(secondPoint)
            userEvent.click(thirdPoint)

            assert.alwaysCalledWith(openStub, 'https://google.com/search')
            assert.calledThrice(openStub)

            openStub.restore()
        })
    })
})
