/**
 * 生成当前时间戳字符串
 * 格式: YYYY-MM-DDTHH-mm-ss
 * @returns 格式化的时间戳字符串
 */
export const generateTimestamp = (): string => {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  const hours = String(now.getHours()).padStart(2, '0');
  const minutes = String(now.getMinutes()).padStart(2, '0');
  const seconds = String(now.getSeconds()).padStart(2, '0');
  return `${year}-${month}-${day}T${hours}-${minutes}-${seconds}`;
};

/**
 * 生成用于文件名的安全时间戳
 * 确保在所有操作系统上都是有效的文件名
 * @returns 安全的文件名时间戳
 */
export const generateFileTimestamp = (): string => {
  return generateTimestamp();
}; 