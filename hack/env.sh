#export LLAMA_STACK_MODEL="meta-llama/Llama-3.2-3B-Instruct"
#export INFERENCE_MODEL="meta-llama/Llama-3.2-3B-Instruct"
export LLAMA_STACK_PORT=8321
export LLAMA_STACK_SERVER=http://localhost:$LLAMA_STACK_PORT
# Required variables
export LLAMA_STACK_MODEL="gemini-1.5-flash"  # or "gemini-1.5-pro" for newer models
export INFERENCE_MODEL="gemini-1.5-flash"
export GEMINI_API_KEY=""  # Get this from Google AI Studio

# Optional configuration
export LLAMA_STACK_MAX_TOKENS=2048
export LLAMA_STACK_TEMPERATURE=0.7
