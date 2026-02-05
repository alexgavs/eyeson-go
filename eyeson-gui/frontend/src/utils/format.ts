export const formatDate = (dateValue: string | number) => {
  if (!dateValue) return '-';
  try {
    if (typeof dateValue === 'number') {
      return new Date(dateValue * 1000).toLocaleString();
    }
    if (typeof dateValue === 'string' && /^\d+$/.test(dateValue)) {
      return new Date(parseInt(dateValue) * 1000).toLocaleString();
    }
    return new Date(dateValue).toLocaleString();
  } catch {
    return String(dateValue);
  }
};
