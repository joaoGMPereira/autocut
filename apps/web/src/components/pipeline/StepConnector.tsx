'use client';

interface StepConnectorProps {
  status: 'idle' | 'active';
  label?: string;
}

export function StepConnector({ status, label }: StepConnectorProps) {
  return (
    <div className="flex flex-col items-center py-1">
      <div
        className={`w-px h-8 ${
          status === 'active'
            ? 'bg-[#00D4FF]'
            : 'border-l border-dashed border-[#5C5C80]'
        }`}
      />
      {label && (
        <span className={`text-[10px] mt-1 ${
          status === 'active' ? 'text-[#00D4FF]' : 'text-[#5C5C80]'
        }`}>
          {label}
        </span>
      )}
    </div>
  );
}
