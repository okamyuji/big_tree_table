interface DateRangeFilterProps {
  dateFrom: string;
  dateTo: string;
  onDateFromChange: (value: string) => void;
  onDateToChange: (value: string) => void;
}

export function DateRangeFilter({
  dateFrom,
  dateTo,
  onDateFromChange,
  onDateToChange,
}: DateRangeFilterProps) {
  return (
    <fieldset className="flex flex-col gap-1">
      <legend className="text-sm font-medium text-gray-700">注文日</legend>
      <div className="flex gap-2 items-center">
        <label htmlFor="date-from" className="text-sm">
          開始
        </label>
        <input
          id="date-from"
          type="date"
          value={dateFrom}
          onChange={(e) => onDateFromChange(e.target.value)}
          className="border border-gray-300 rounded px-2 py-1 text-sm"
        />
        <label htmlFor="date-to" className="text-sm">
          終了
        </label>
        <input
          id="date-to"
          type="date"
          value={dateTo}
          onChange={(e) => onDateToChange(e.target.value)}
          className="border border-gray-300 rounded px-2 py-1 text-sm"
        />
      </div>
    </fieldset>
  );
}
