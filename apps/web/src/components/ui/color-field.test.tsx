import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ColorField } from './color-field'

describe('ColorField', () => {
  it('renders a swatch button for each default token', () => {
    render(<ColorField value="#000000" onValueChange={vi.fn()} />)
    // 8 default tokens
    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(8)
  })

  it('renders only specified tokens when tokens prop provided', () => {
    render(
      <ColorField
        value="#000000"
        onValueChange={vi.fn()}
        tokens={['brand', 'success']}
      />,
    )
    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(2)
    expect(screen.getByRole('button', { name: 'Brand' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Success' })).toBeInTheDocument()
  })

  it('clicking a swatch calls onValueChange with hex and token', () => {
    const onValueChange = vi.fn()
    render(<ColorField value="#000000" onValueChange={onValueChange} tokens={['brand']} />)
    fireEvent.click(screen.getByRole('button', { name: 'Brand' }))
    expect(onValueChange).toHaveBeenCalledWith('#C6F24E', 'brand')
  })

  it('selected token swatch has aria-pressed true, others false', () => {
    render(<ColorField value="#C6F24E" onValueChange={vi.fn()} tokens={['brand', 'success']} />)
    const brandButton = screen.getByRole('button', { name: 'Brand' })
    const successButton = screen.getByRole('button', { name: 'Success' })
    expect(brandButton).toHaveAttribute('aria-pressed', 'true')
    expect(successButton).toHaveAttribute('aria-pressed', 'false')
  })

  it('renders label when label prop provided', () => {
    render(<ColorField value="#000000" onValueChange={vi.fn()} label="Cor do Texto" />)
    expect(screen.getByText('Cor do Texto')).toBeInTheDocument()
  })

  it('data-slot="color-field" present on root', () => {
    render(<ColorField value="#000000" onValueChange={vi.fn()} />)
    const el = document.querySelector('[data-slot="color-field"]')
    expect(el).not.toBeNull()
  })

  it('className merges on root', () => {
    render(<ColorField value="#000000" onValueChange={vi.fn()} className="extra-class" />)
    const el = document.querySelector('[data-slot="color-field"]')!
    expect(el.className).toContain('extra-class')
  })
})
